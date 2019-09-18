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
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"golang.org/x/xerrors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/rand"
)

// uidMatch determines the uid variable within the resource template
var uidMatch = []byte(`$(uid)`)

type ResolvedBinding struct {
	TriggerBinding  *triggersv1.TriggerBinding
	TriggerTemplate *triggersv1.TriggerTemplate
}

type getTriggerBinding func(name string, options metav1.GetOptions) (*triggersv1.TriggerBinding, error)
type getTriggerTemplate func(name string, options metav1.GetOptions) (*triggersv1.TriggerTemplate, error)

func ResolveBinding(trigger triggersv1.EventListenerTrigger, getTB getTriggerBinding, getTT getTriggerTemplate) (ResolvedBinding, error) {
	tbName := trigger.Binding.Name
	tb, err := getTB(tbName, metav1.GetOptions{})
	if err != nil {
		return ResolvedBinding{}, xerrors.Errorf("Error getting TriggerBinding %s: %s", tbName, err)
	}
	ttName := trigger.Template.Name
	tt, err := getTT(ttName, metav1.GetOptions{})
	if err != nil {
		return ResolvedBinding{}, xerrors.Errorf("Error getting TriggerTemplate %s: %s", ttName, err)
	}
	return ResolvedBinding{TriggerBinding: tb, TriggerTemplate: tt}, nil
}

// MergeInDefaultParams returns the params with the addition of all
// paramSpecs that have default values and are already in the params list
func MergeInDefaultParams(params []pipelinev1.Param, paramSpecs []pipelinev1.ParamSpec) []pipelinev1.Param {
	allParamsMap := map[string]pipelinev1.ArrayOrString{}
	for _, paramSpec := range paramSpecs {
		if paramSpec.Default != nil {
			allParamsMap[paramSpec.Name] = *paramSpec.Default
		}
	}
	for _, param := range params {
		allParamsMap[param.Name] = param.Value
	}
	allParams := make([]pipelinev1.Param, len(allParamsMap))
	i := 0
	for name, value := range allParamsMap {
		allParams[i] = pipelinev1.Param{Name: name, Value: value}
		i++
	}
	return allParams
}

// ApplyParamsToResourceTemplate returns the TriggerResourceTemplate with the
// param values substituted for all matching param variables in the template
func ApplyParamsToResourceTemplate(params []pipelinev1.Param, rt json.RawMessage) json.RawMessage {
	// Assume the params are valid
	for _, param := range params {
		rt = applyParamToResourceTemplate(param, rt)
	}
	return rt
}

// applyParamToResourceTemplate returns the TriggerResourceTemplate with the
// param value substituted for all matching param variables in the template
func applyParamToResourceTemplate(param pipelinev1.Param, rt json.RawMessage) json.RawMessage {
	// Assume the param is valid
	paramVariable := fmt.Sprintf("$(params.%s)", param.Name)
	return bytes.Replace(rt, []byte(paramVariable), []byte(param.Value.StringVal), -1)
}

// ApplyInputParamsToOutputParams returns the outputParams with substituted
// values for every matching inputParam
func ApplyInputParamsToOutputParams(inputParams []pipelinev1.Param, outputParams []pipelinev1.Param) []pipelinev1.Param {
	// Assume the params are valid
	for _, inputParam := range inputParams {
		for i := range outputParams {
			intputParamVariable := fmt.Sprintf("$(inputParams.%s)", inputParam.Name)
			outputParams[i].Value.StringVal = strings.Replace(outputParams[i].Value.StringVal, intputParamVariable, inputParam.Value.StringVal, -1)
		}
	}
	return outputParams
}

// Uid generates a random string like the Kubernetes apiserver generateName metafield postfix.
func Uid() string {
	return rand.String(5)
}

// ApplyUIDToResourceTemplate returns the TriggerResourceTemplate after uid replacement
// The same uid should be used per trigger to properly address resources throughout the TriggerTemplate.
func ApplyUIDToResourceTemplate(rt json.RawMessage, uid string) json.RawMessage {
	return bytes.Replace(rt, uidMatch, []byte(uid), -1)
}
