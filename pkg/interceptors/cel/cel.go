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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter/functions"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

// Interceptor implements a CEL based interceptor, that uses CEL expressions
// against the incoming body and headers to match, if the expression returns
// a true value, then the interception is "successful".
type Interceptor struct {
	Logger *zap.SugaredLogger
	CEL    *triggersv1.CELInterceptor
}

// NewInterceptor creates a prepopulated Interceptor.
func NewInterceptor(cel *triggersv1.CELInterceptor, l *zap.SugaredLogger) interceptors.Interceptor {
	return &Interceptor{
		Logger: l,
		CEL:    cel,
	}
}

// ExecuteTrigger is an implementation of the Interceptor interface.
func (w *Interceptor) ExecuteTrigger(request *http.Request) (*http.Response, error) {
	env, err := makeCelEnv()
	if err != nil {
		return nil, fmt.Errorf("error creating cel environment: %w", err)
	}

	var payload = []byte(`{}`)
	if request.Body != nil {
		defer request.Body.Close()
		payload, err = ioutil.ReadAll(request.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading request body: %w", err)
		}
	}

	evalContext, err := makeEvalContext(payload, request)
	if err != nil {
		return nil, fmt.Errorf("error making the evaluation context: %w", err)
	}

	out, err := evaluate(w.CEL.Filter, env, evalContext)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression '%s': %w", w.CEL.Filter, err)
	}

	if out != types.True {
		return nil, fmt.Errorf("expression %s did not return true", w.CEL.Filter)

	}

	for _, u := range w.CEL.Overlays {
		val, err := evaluate(u.Expression, env, evalContext)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate overlay expression '%s': %w", w.CEL.Filter, err)
		}

		b, ok := val.Value().([]byte)
		if !ok {
			return nil, fmt.Errorf("failed to convert overlay result to bytes: %w", err)
		}

		payload, err = sjson.SetRawBytes(payload, u.Key, b)
		if err != nil {
			return nil, fmt.Errorf("failed to sjson for key '%s' to '%s': %w", u.Key, val, err)
		}
	}

	return &http.Response{
		Header: request.Header,
		Body:   ioutil.NopCloser(bytes.NewBuffer(payload)),
	}, nil

}

func evaluate(expr string, env cel.Env, data map[string]interface{}) (ref.Val, error) {
	parsed, issues := env.Parse(expr)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	checked, issues := env.Check(parsed)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	prg, err := env.Program(checked, embeddedFunctions())
	if err != nil {
		return nil, err
	}

	out, _, err := prg.Eval(data)
	return out, err
}

func embeddedFunctions() cel.ProgramOption {
	return cel.Functions(
		&functions.Overload{
			Operator: "match",
			Function: matchHeader,
		},
		&functions.Overload{
			Operator: "truncate",
			Binary:   truncateString,
		},
		&functions.Overload{
			Operator: "base64",
			Unary:    base64Decode,
		},
	)

}
func makeCelEnv() (cel.Env, error) {
	mapStrDyn := decls.NewMapType(decls.String, decls.Dyn)
	return cel.NewEnv(
		cel.Declarations(
			decls.NewIdent("body", mapStrDyn, nil),
			decls.NewIdent("header", mapStrDyn, nil),
			decls.NewFunction("match",
				decls.NewInstanceOverload("match_map_string_string",
					[]*exprpb.Type{mapStrDyn, decls.String, decls.String}, decls.Bool)),
			decls.NewFunction("truncate",
				decls.NewOverload("truncate_string_uint",
					[]*exprpb.Type{decls.String, decls.Int}, decls.String)),
			decls.NewFunction("base64",
				decls.NewOverload("base64_string", []*exprpb.Type{decls.String}, decls.String)),
		))
}

func makeEvalContext(body []byte, r *http.Request) (map[string]interface{}, error) {
	var jsonMap map[string]interface{}
	err := json.Unmarshal(body, &jsonMap)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"body": jsonMap, "header": r.Header}, nil

}

func matchHeader(vals ...ref.Val) ref.Val {
	h, err := vals[0].ConvertToNative(reflect.TypeOf(http.Header{}))
	if err != nil {
		return types.NewErr("failed to convert to http.Header: %w", err)
	}

	key, ok := vals[1].(types.String)
	if !ok {
		return types.ValOrErr(key, "unexpected type '%v' passed to match", vals[1].Type())
	}

	val, ok := vals[2].(types.String)
	if !ok {
		return types.ValOrErr(val, "unexpected type '%v' passed to match", vals[2].Type())
	}

	return types.Bool(h.(http.Header).Get(string(key)) == string(val))

}

func truncateString(lhs, rhs ref.Val) ref.Val {
	str, ok := lhs.(types.String)
	if !ok {
		return types.ValOrErr(str, "unexpected type '%v' passed to truncate", lhs.Type())
	}

	n, ok := rhs.(types.Int)
	if !ok {
		return types.ValOrErr(n, "unexpected type '%v' passed to truncate", rhs.Type())
	}

	return types.Bytes([]byte(str[:max(n, types.Int(len(str)))]))
}

func max(x, y types.Int) types.Int {
	switch x.Compare(y) {
	case types.IntNegOne:
		return x
	case types.IntOne:
		return y
	default:
		return x
	}
}

func base64Decode(val ref.Val) ref.Val {
	in, ok := val.(types.String)
	if !ok {
		return types.ValOrErr(val, "unexpected type '%v' passed to base64", val.Type())
	}

	out, err := base64.StdEncoding.DecodeString(string(in))
	if err != nil {
		return types.ValOrErr(val, "unable to decode string: %v", err)
	}
	fmt.Println("Decoded body", string(out))

	return types.Bytes(out)
}
