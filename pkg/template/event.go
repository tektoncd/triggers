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

package template

import (
	"encoding/json"
	"regexp"
	"strings"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tidwall/gjson"
	"golang.org/x/xerrors"
)

// bodyPathVarRegex determines valid body path variables
// The body regular expression allows for a subset of GJSON syntax, the mininum
// required to navigate through dictionaries, query arrays and support
// namespaced label names e.g. tekton.dev/eventlistener
var bodyPathVarRegex = regexp.MustCompile(`\$\(body(\.[[:alnum:]/_\-\.\\]+|\.#\([[:alnum:]=<>%!"\*_-]+\)#??)*\)`)

// The headers regular expression allows for simple navigation down a hierarchy
// of dictionaries
var headerVarRegex = regexp.MustCompile(`\$\(header(\.[[:alnum:]_\-]+)?\)`)

// getBodyPathFromVar returns the body path given an body path variable
// $(body.my.path) -> my.path
// $(body) returns an empty string "" because there is no body path
func getBodyPathFromVar(bodyPathVar string) string {
	// Assume bodyPathVar matches the bodyPathVarRegex
	if bodyPathVar == "$(body)" {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(bodyPathVar, "$(body."), ")")
}

// getHeaderFromVar returns the header given a header variable
// $(header.example) -> example
func getHeaderFromVar(headerVar string) string {
	// Assume headerVar matches the headerVarRegex
	if headerVar == "$(header)" {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(headerVar, "$(header."), ")")
}

// ApplyBodyToParams returns the params with each body path variable replaced
// with the appropriate data from the body. Returns an error when the body
// path variable is not found in the body.
func ApplyBodyToParams(body []byte, params []pipelinev1.Param) ([]pipelinev1.Param, error) {
	for i := range params {
		param, err := applyBodyToParam(body, params[i])
		if err != nil {
			return nil, err
		}
		params[i] = param
	}
	return params, nil
}

// applyBodyToParam returns the param with each body path variable replaced
// with the appropriate data from the body. Returns an error when the body
// path variable is not found in the body.
func applyBodyToParam(body []byte, param pipelinev1.Param) (pipelinev1.Param, error) {
	// Get each body path variable in the param
	bodyPathVars := bodyPathVarRegex.FindAllString(param.Value.StringVal, -1)
	for _, bodyPathVar := range bodyPathVars {
		bodyPath := getBodyPathFromVar(bodyPathVar)
		bodyPathValue, err := getBodyPathValue(body, bodyPath)
		if err != nil {
			return param, err
		}
		param.Value.StringVal = strings.Replace(param.Value.StringVal, bodyPathVar, bodyPathValue, -1)
	}
	return param, nil
}

// getBodyPathValue returns the value of the bodyPath in the body. An error
// is returned if the bodyPath is not found in the body.
func getBodyPathValue(body []byte, bodyPath string) (string, error) {
	var bodyPathValue string
	if bodyPath == "" {
		// $(body) has an empty bodyPath, so use the entire body as the bodyValue
		bodyPathValue = string(body)
	} else {
		bodyPathResult := gjson.GetBytes(body, bodyPath)
		if bodyPathResult.Index == 0 {
			return "", xerrors.Errorf("Error body path %s not found in the body %s", bodyPath, string(body))
		}
		bodyPathValue = bodyPathResult.String()
		if bodyPathResult.Type == gjson.Null {
			bodyPathValue = "null"
		}
	}
	return strings.Replace(bodyPathValue, `"`, `\"`, -1), nil
}

// ApplyHeaderToParams returns the params with each header variable replaced
// with the appropriate header value. Returns an error when the header variable
// is not found.
func ApplyHeaderToParams(header map[string][]string, params []pipelinev1.Param) ([]pipelinev1.Param, error) {
	for i := range params {
		param, err := applyHeaderToParam(header, params[i])
		if err != nil {
			return nil, err
		}
		params[i] = param
	}
	return params, nil
}

// applyHeaderToParam returns the param with each header variable replaced
// with the appropriate header value. Returns an error when the header variable
// is not found.
func applyHeaderToParam(header map[string][]string, param pipelinev1.Param) (pipelinev1.Param, error) {
	// Get each header variable in the param
	headerVars := headerVarRegex.FindAllString(param.Value.StringVal, -1)
	for _, headerVar := range headerVars {
		headerName := getHeaderFromVar(headerVar)
		headerValue, err := getHeaderValue(header, headerName)
		if err != nil {
			return param, err
		}
		param.Value.StringVal = strings.Replace(param.Value.StringVal, headerVar, headerValue, -1)
	}
	return param, nil
}

// getHeaderValue returns a string representation of the headerName in the event
// header. An error is returned if the headerName is not found in the header.
func getHeaderValue(header map[string][]string, headerName string) (string, error) {
	var headerValue string
	if headerName == "" {
		// $(header) has an empty headerName, so use all the headers in the headerValue
		b, err := json.Marshal(&header)
		if err != nil {
			return "", xerrors.Errorf("Error marshalling header %s: %s", header, err)
		}
		headerValue = string(b)
	} else {
		value, ok := header[headerName]
		if !ok {
			return "", xerrors.Errorf("Error headerName %s not found in the event header %s", headerName, header)
		}
		headerValue = strings.Join(value, " ")
	}
	return strings.Replace(headerValue, `"`, `\"`, -1), nil
}

// NewResources returns all resources defined when applying the event and
// elParams to the TriggerTemplate and TriggerBinding in the ResolvedBinding.
func NewResources(body []byte, header map[string][]string, elParams []pipelinev1.Param, binding ResolvedBinding) ([]json.RawMessage, error) {

	params, err := mergeBindingParams(binding.TriggerBindings)
	if err != nil {
		return []json.RawMessage{}, xerrors.Errorf("error merging TriggerBinding params: %v", err)
	}

	params, err = ApplyBodyToParams(body, params)
	if err != nil {
		return []json.RawMessage{}, xerrors.Errorf("Error applying body to TriggerBinding params: %s", err)
	}
	params, err = ApplyHeaderToParams(header, params)
	if err != nil {
		return []json.RawMessage{}, xerrors.Errorf("Error applying header to TriggerBinding params: %s", err)
	}
	params, err = MergeParams(params, elParams)
	if err != nil {
		return []json.RawMessage{}, xerrors.Errorf("Error merging params from EventListener with TriggerBinding params: %s", err)
	}
	params = MergeInDefaultParams(params, binding.TriggerTemplate.Spec.Params)

	resources := make([]json.RawMessage, len(binding.TriggerTemplate.Spec.ResourceTemplates))
	uid := UID()
	for i := range binding.TriggerTemplate.Spec.ResourceTemplates {
		resources[i] = ApplyParamsToResourceTemplate(params, binding.TriggerTemplate.Spec.ResourceTemplates[i].RawMessage)
		resources[i] = ApplyUIDToResourceTemplate(resources[i], uid)
	}
	return resources, nil
}

func mergeBindingParams(bindings []*triggersv1.TriggerBinding) ([]pipelinev1.Param, error) {
	var params []pipelinev1.Param

	for _, b := range bindings {
		var err error
		params, err = MergeParams(params, b.Spec.Params)
		if err != nil {
			return nil, xerrors.Errorf("error merging params: %v", err)
		}
	}

	return params, nil
}
