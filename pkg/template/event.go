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
	"github.com/tidwall/gjson"
	"golang.org/x/xerrors"
)

// eventPathVarRegex determines valid event path variables
var eventPathVarRegex = regexp.MustCompile(`\$\(event(.[0-9A-Za-z_-]+)*\)`)

// getEventPathFromVar returns the event path given an event path variable
// $(event.my.path) -> my.path
// $(event) returns an empty string "" because there is no event path
func getEventPathFromVar(eventPathVar string) string {
	// Assume eventPathVar matches the eventPathVarRegex
	if eventPathVar == "$(event)" {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(eventPathVar, "$(event."), ")")
}

// ApplyEventToOutputParams returns the params with each event path variable replaced
// with the appropriate data from the event. Returns an error when the event
// path variable is not found in the event.
func ApplyEventToOutputParams(event []byte, params []pipelinev1.Param) ([]pipelinev1.Param, error) {
	for i := range params {
		param, err := applyEventToParam(event, params[i])
		if err != nil {
			return nil, err
		}
		params[i] = param
	}
	return params, nil
}

// applyEventToParam returns the param with each event path variable replaced
// with the appropriate data from the event. Returns an error when the event
// path variable is not found in the event.
func applyEventToParam(event []byte, param pipelinev1.Param) (pipelinev1.Param, error) {
	// Get each event path variable in the param
	eventPathVars := eventPathVarRegex.FindAllString(param.Value.StringVal, -1)
	for _, eventPathVar := range eventPathVars {
		eventPath := getEventPathFromVar(eventPathVar)
		eventPathValue, err := getEventPathValue(event, eventPath)
		if err != nil {
			return param, err
		}
		param.Value.StringVal = strings.Replace(param.Value.StringVal, eventPathVar, eventPathValue, -1)
	}
	return param, nil
}

// getEventPathValue returns the value of the eventPath in the event. An error
// is returned if the eventPath is not found in the event.
func getEventPathValue(event []byte, eventPath string) (string, error) {
	var eventPathValue string
	if eventPath == "" {
		// $(event) has an empty eventPath, so use the entire event as the eventValue
		eventPathValue = string(event)
	} else {
		// eventPathValue = gjson.GetBytes(event, eventPath).String()
		eventPathResult := gjson.GetBytes(event, eventPath)
		if eventPathResult.Index == 0 {
			return "", xerrors.Errorf("Error event path %s not found in the event %s", eventPath, string(event))
		}
		eventPathValue = eventPathResult.String()
		if eventPathResult.Type == gjson.Null {
			eventPathValue = "null"
		}
	}
	return eventPathValue, nil
}

// NewResources returns all resources defined when applying the event and
// elParams to the TriggerTemplate and TriggerBinding in the ResolvedBinding.
func NewResources(event []byte, elParams []pipelinev1.Param, binding ResolvedBinding) ([]json.RawMessage, error) {
	inputParams := MergeInDefaultParams(elParams, binding.TriggerBinding.Spec.InputParams)
	outputParams := ApplyInputParamsToOutputParams(inputParams, binding.TriggerBinding.Spec.OutputParams)
	params, err := ApplyEventToOutputParams(event, outputParams)
	if err != nil {
		return []json.RawMessage{}, xerrors.Errorf("Error applying event to TriggerBinding outputParams: %s", err)
	}
	params = MergeInDefaultParams(params, binding.TriggerTemplate.Spec.Params)

	resources := make([]json.RawMessage, len(binding.TriggerTemplate.Spec.ResourceTemplates))
	uid := Uid()
	for i := range binding.TriggerTemplate.Spec.ResourceTemplates {
		resources[i] = ApplyParamsToResourceTemplate(params, binding.TriggerTemplate.Spec.ResourceTemplates[i].RawMessage)
		resources[i] = ApplyUIDToResourceTemplate(resources[i], uid)
	}
	return resources, nil
}
