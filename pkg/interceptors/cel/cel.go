/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/tektoncd/triggers/pkg/interceptors"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	celext "github.com/google/cel-go/ext"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"k8s.io/client-go/kubernetes"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

var _ triggersv1.InterceptorInterface = (*Interceptor)(nil)

// Interceptor implements a CEL based interceptor that uses CEL expressions
// against the incoming body and headers to match, if the expression returns
// a true value, then the interception is "successful".
type Interceptor struct {
	KubeClientSet    kubernetes.Interface
	Logger           *zap.SugaredLogger
	CEL              *triggersv1.CELInterceptor
	TriggerNamespace string
}

var (
	structType = reflect.TypeOf(&structpb.Value{})
	listType   = reflect.TypeOf(&structpb.ListValue{})
	mapType    = reflect.TypeOf(&structpb.Struct{})
)

// NewInterceptor creates a prepopulated Interceptor.
func NewInterceptor(k kubernetes.Interface, l *zap.SugaredLogger) *Interceptor {
	return &Interceptor{
		Logger:        l,
		KubeClientSet: k,
	}
}

func evaluate(expr string, env *cel.Env, data map[string]interface{}) (ref.Val, error) {
	parsed, issues := env.Parse(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to parse expression %#v: %w", expr, issues.Err())
	}

	checked, issues := env.Check(parsed)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("expression %#v check failed: %w", expr, issues.Err())
	}

	prg, err := env.Program(checked)
	if err != nil {
		return nil, fmt.Errorf("expression %#v failed to create a Program: %w", expr, err)
	}

	out, _, err := prg.Eval(data)
	if err != nil {
		return nil, fmt.Errorf("expression %#v failed to evaluate: %w", expr, err)
	}
	return out, nil
}

func makeCelEnv(ns string, k kubernetes.Interface) (*cel.Env, error) {
	mapStrDyn := decls.NewMapType(decls.String, decls.Dyn)
	return cel.NewEnv(
		Triggers(ns, k),
		celext.Strings(),
		celext.Encoders(),
		cel.Declarations(
			decls.NewVar("body", mapStrDyn),
			decls.NewVar("header", mapStrDyn),
			decls.NewVar("extensions", mapStrDyn),
			decls.NewVar("requestURL", decls.String),
		))
}

func makeEvalContext(body []byte, h http.Header, url string, extensions map[string]interface{}) (map[string]interface{}, error) {
	var jsonMap map[string]interface{}
	err := json.Unmarshal(body, &jsonMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the body as JSON: %w", err)
	}
	return map[string]interface{}{
		"body":       jsonMap,
		"header":     h,
		"requestURL": url,
		"extensions": extensions,
	}, nil
}

func (w *Interceptor) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	p := triggersv1.CELInterceptor{}
	if err := interceptors.UnmarshalParams(r.InterceptorParams, &p); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to parse interceptor params: %v", err)
	}

	ns, _ := triggersv1.ParseTriggerID(r.Context.TriggerID)
	env, err := makeCelEnv(ns, w.KubeClientSet)
	if err != nil {
		return interceptors.Failf(codes.Internal, "error creating cel environment: %v", err)
	}

	var payload = []byte(`{}`)
	if r.Body != "" {
		payload = []byte(r.Body)
	}

	evalContext, err := makeEvalContext(payload, r.Header, r.Context.EventURL, r.Extensions)
	if err != nil {
		return interceptors.Failf(codes.InvalidArgument, "error making the evaluation context: %v", err)
	}

	if p.Filter != "" {
		out, err := evaluate(p.Filter, env, evalContext)

		if err != nil {
			return interceptors.Failf(codes.InvalidArgument, "error evaluating cel expression: %v", err)
		}

		if out != types.True {
			return interceptors.Failf(codes.FailedPrecondition, "expression %s did not return true", p.Filter)
		}
	}

	// Empty JSON body bytes.
	// We use []byte instead of map[string]interface{} to allow ovewriting keys using sjson.
	var extensions []byte
	for _, u := range p.Overlays {
		val, err := evaluate(u.Expression, env, evalContext)
		if err != nil {
			return interceptors.Failf(codes.InvalidArgument, "error evaluating cel expression: %v", err)
		}

		var raw interface{}
		var b []byte

		switch val.(type) {
		// this causes types.Bytes to be rendered as a Base64 string this is
		// because the Go JSON Encoder encodes []bytes this way, see
		// https://golang.org/pkg/encoding/json/#Marshal
		//
		// An alternative might be to return " + val + " for types.Bytes to
		// simulate the the JSON encoding.
		case types.String, types.Bytes:
			raw, err = val.ConvertToNative(structType)
			if err == nil {
				b, err = raw.(*structpb.Value).MarshalJSON()
			}
		case types.Double, types.Int:
			raw, err = val.ConvertToNative(structType)
			if err == nil {
				b, err = raw.(*structpb.Value).MarshalJSON()
			}
		case traits.Lister:
			raw, err = val.ConvertToNative(listType)
			if err == nil {
				s, err := protojson.Marshal(raw.(proto.Message))
				if err == nil {
					b = s
				}
			}
		case traits.Mapper:
			raw, err = val.ConvertToNative(mapType)
			if err == nil {
				s, err := protojson.Marshal(raw.(proto.Message))
				if err == nil {
					b = s
				}
			}
		case types.Bool:
			raw, err = val.ConvertToNative(structType)
			if err == nil {
				b, err = json.Marshal(raw.(*structpb.Value).GetBoolValue())
			}
		default:
			raw, err = val.ConvertToNative(reflect.TypeOf([]byte{}))
			if err == nil {
				b = raw.([]byte)
			}
		}

		if err != nil {
			return interceptors.Failf(codes.Internal, "failed to convert overlay result to type: %v", err)
		}

		// TODO: For backwards compatibility, consider also merging and returning the body back?
		if extensions == nil {
			extensions = []byte("{}")
		}
		extensions, err = sjson.SetRawBytes(extensions, u.Key, b)
		if err != nil {
			return interceptors.Failf(codes.Internal, "failed to sjson for key '%s' to '%s': %v", u.Key, val, err)
		}
	}

	if extensions == nil {
		return &triggersv1.InterceptorResponse{
			Continue: true,
		}
	}

	extensionsMap := map[string]interface{}{}
	if err := json.Unmarshal(extensions, &extensionsMap); err != nil {
		return interceptors.Failf(codes.Internal, "failed to unmarshal extensions into map: %v", err)
	}
	return &triggersv1.InterceptorResponse{
		Continue:   true,
		Extensions: extensionsMap,
	}
}
