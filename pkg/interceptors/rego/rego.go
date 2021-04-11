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

package rego

import (
	"context"
	"encoding/json"
	"fmt"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"google.golang.org/grpc/codes"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

const (
	QueryParam   = "query"
	ExtensionKey = "extension"
	Overlays     = "overlays"
	Bindings     = "bindings"
	Single       = "single"
)

var _ triggersv1.InterceptorInterface = (*Interceptor)(nil)

type Interceptor struct {
	Logger        *zap.SugaredLogger
	KubeClientSet kubernetes.Interface
}

func NewInterceptor(k kubernetes.Interface, l *zap.SugaredLogger) *Interceptor {
	return &Interceptor{
		Logger:        l,
		KubeClientSet: k,
	}
}

func (w *Interceptor) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {

	var body map[string]interface{}

	if err := json.Unmarshal([]byte(r.Body), &body); err != nil {
		return errorResponse(fmt.Errorf("unable to marshal body to json: %w", err))
	}

	inputBody := map[string]interface{}{
		"body":       body,
		"header":     r.Header,
		"extensions": r.Extensions,
	}

	query := r.InterceptorParams[QueryParam].(string)
	module := fmt.Sprintf(`package tekton
	default filter = false
	%s
	`, query)
	compiler, err := ast.CompileModules(map[string]string{
		"tekton.rego": module,
	})

	if err != nil {
		return errorResponse(err)
	}
	regoFilter := rego.New(
		rego.Query("data.tekton.filter"),
		rego.Compiler(compiler),
		rego.Input(inputBody))

	rs, err := regoFilter.Eval(ctx)
	if err != nil {
		return &triggersv1.InterceptorResponse{
			Continue: false,
			Status: triggersv1.Status{
				Message: err.Error(),
				Code:    codes.Aborted,
			},
		}
	}
	// filter query failed
	if !rs[0].Expressions[0].Value.(bool) {
		return &triggersv1.InterceptorResponse{
			Continue: false,
			Status: triggersv1.Status{
				Code:    codes.FailedPrecondition,
				Message: fmt.Sprintf("%s unmatched", query),
			},
		}
	}
	extensions := make(map[string]interface{})

	// To add keys to an extension, you provide a query and a set of variable to bind to
	// The query is in the `query` key
	// The extension key set is in `extension`
	// The rego result variables to look for is a list of strings called `bindings`
	// setting `single` to true means that we will expect a single value in the result set
	// for example:
	/*
		overlays:
		- extension: my-ext
		  query: input.body.users[idx].id = user_id
		  bindings:
		  - user_id
	*/
	if b, ok := r.InterceptorParams[Overlays]; ok {
		for _, bindingMap := range b.([]map[string]interface{}) {
			extName := bindingMap[ExtensionKey].(string)
			result, err := createResultArray(ctx, rego.New(
				rego.Query(bindingMap[QueryParam].(string)),
				rego.Compiler(compiler),
				rego.Input(inputBody)), bindingMap)
			if err != nil {
				return errorResponse(fmt.Errorf("unable to evalutate extension for %s: %w", extName, err))
			}
			if result != nil {
				extensions[extName] = result
			}
		}
	}
	return &triggersv1.InterceptorResponse{
		Continue:   true,
		Extensions: extensions,
	}
}

func errorResponse(err error) *triggersv1.InterceptorResponse {
	return &triggersv1.InterceptorResponse{
		Continue: false,
		Status: triggersv1.Status{
			Message: err.Error(),
			Code:    codes.Aborted,
		},
	}
}

func createResultArray(ctx context.Context, r *rego.Rego, bindingMap map[string]interface{}) (map[string]interface{}, error) {
	rs, err := r.Eval(ctx)
	if err != nil {
		return nil, err
	}
	if len(rs) == 0 {
		return nil, nil
	}
	result := make(map[string]interface{})
	bindings := bindingMap[Bindings].([]string)
	for _, b := range bindings {
		resultArray := make([]interface{}, len(rs))
		for i := range rs {
			resultArray[i] = rs[i].Bindings[b]
		}
		result[b] = resultArray
	}
	if single, ok := bindingMap[Single].(bool); ok && single {
		for k, v := range result {
			result[k] = v.([]interface{})[0]
		}
	}
	return result, nil
}
