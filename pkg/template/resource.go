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

	"github.com/google/uuid"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// uidMatch determines the uid variable within the resource template
var uidMatch = []byte(`$(uid)`)

// ResolvedTrigger contains the dereferenced TriggerBindings and
// TriggerTemplate after resolving the k8s ObjectRef.
type ResolvedTrigger struct {
	TriggerBindings        []*triggersv1.TriggerBinding
	ClusterTriggerBindings []*triggersv1.ClusterTriggerBinding
	TriggerTemplate        *triggersv1.TriggerTemplate
	BindingParams          []triggersv1.Param
}

type getTriggerBinding func(name string) (*triggersv1.TriggerBinding, error)
type getTriggerTemplate func(name string) (*triggersv1.TriggerTemplate, error)
type getClusterTriggerBinding func(name string) (*triggersv1.ClusterTriggerBinding, error)

// ResolveTrigger takes in a trigger containing object refs to bindings and
// templates and resolves them to their underlying values.
func ResolveTrigger(trigger triggersv1.Trigger, getTB getTriggerBinding, getCTB getClusterTriggerBinding, getTT getTriggerTemplate) (ResolvedTrigger, error) {
	bp, err := resolveBindingsToParams(trigger.Spec.Bindings, getTB, getCTB)
	if err != nil {
		return ResolvedTrigger{}, fmt.Errorf("failed to resolve bindings: %w", err)
	}

	var resolvedTT *triggersv1.TriggerTemplate
	if trigger.Spec.Template.Spec != nil {
		resolvedTT = &triggersv1.TriggerTemplate{
			ObjectMeta: metav1.ObjectMeta{}, // Unused. TODO: Just return Specs from here.
			Spec:       *trigger.Spec.Template.Spec,
		}
	} else {
		var ttName string
		if trigger.Spec.Template.Ref != nil {
			ttName = *trigger.Spec.Template.Ref
		}
		resolvedTT, err = getTT(ttName)
		if err != nil {
			return ResolvedTrigger{}, fmt.Errorf("error getting TriggerTemplate %s: %w", ttName, err)
		}
	}

	return ResolvedTrigger{TriggerTemplate: resolvedTT, BindingParams: bp}, nil
}

// resolveBindingsToParams takes in both embedded bindings and references and returns a list of resolved Param values.ResolveBindingsToParams
func resolveBindingsToParams(bindings []*triggersv1.TriggerSpecBinding, getTB getTriggerBinding, getCTB getClusterTriggerBinding) ([]triggersv1.Param, error) {
	bindingParams := []triggersv1.Param{}
	for _, b := range bindings {
		switch {
		case b.Name != "" && b.Value != nil:
			bindingParams = append(bindingParams, triggersv1.Param{
				Name:  b.Name,
				Value: *b.Value,
			})

		case b.Ref != "" && b.Kind == triggersv1.ClusterTriggerBindingKind:
			ctb, err := getCTB(b.Ref)
			if err != nil {
				return nil, fmt.Errorf("error getting ClusterTriggerBinding %s: %w", b.Name, err)
			}
			bindingParams = append(bindingParams, ctb.Spec.Params...)

		case b.Ref != "": // if no kind is set, assume NamespacedTriggerBinding
			tb, err := getTB(b.Ref)
			if err != nil {
				return nil, fmt.Errorf("error getting TriggerBinding %s: %w", b.Name, err)
			}
			bindingParams = append(bindingParams, tb.Spec.Params...)
		default:
			return nil, fmt.Errorf("invalid binding: %v", b)
		}
	}

	// Check for duplicate params
	seen := make(map[string]bool, len(bindingParams))
	for _, p := range bindingParams {
		if seen[p.Name] {
			return nil, fmt.Errorf("duplicate param name: %s", p.Name)
		}
		seen[p.Name] = true
	}
	return bindingParams, nil
}

// applyParamsToResourceTemplate returns the TriggerResourceTemplate with the
// param values substituted for all matching param variables in the template
func applyParamsToResourceTemplate(params []triggersv1.Param, rt json.RawMessage, oldEscape bool) json.RawMessage {
	// Assume the params are valid
	for _, param := range params {
		rt = applyParamToResourceTemplate(param, rt, oldEscape)
	}
	return rt
}

// applyParamToResourceTemplate returns the TriggerResourceTemplate with the
// param value substituted for all matching param variables in the template
func applyParamToResourceTemplate(param triggersv1.Param, rt json.RawMessage, oldEscape bool) json.RawMessage {
	// Assume the param is valid
	paramVariable := fmt.Sprintf("$(tt.params.%s)", param.Name)
	// Escape quotes so that that JSON strings can be appended to regular strings.
	// See #257 for discussion on this behavior.
	if oldEscape {
		paramValue := strings.ReplaceAll(param.Value, `"`, `\"`)
		return bytes.ReplaceAll(rt, []byte(paramVariable), []byte(paramValue))
	}
	return bytes.ReplaceAll(rt, []byte(paramVariable), []byte(param.Value))
}

// UUID generates a Universally Unique IDentifier following RFC 4122.
var UUID = func() string { return uuid.New().String() }

// applyUIDToResourceTemplate returns the TriggerResourceTemplate after uid replacement
// The same uid should be used per trigger to properly address resources throughout the TriggerTemplate.
func applyUIDToResourceTemplate(rt json.RawMessage, uid string) json.RawMessage {
	return bytes.ReplaceAll(rt, uidMatch, []byte(uid))
}

func convertParamMapToArray(paramMap map[string]string) []triggersv1.Param {
	params := []triggersv1.Param{}
	for name, value := range paramMap {
		params = append(params, triggersv1.Param{Name: name, Value: value})
	}
	return params
}

// mergeBindingParams merges params across multiple bindings.
func mergeBindingParams(bindings []*triggersv1.TriggerBinding, clusterbindings []*triggersv1.ClusterTriggerBinding) ([]triggersv1.Param, error) {
	params := []triggersv1.Param{}
	for _, b := range bindings {
		params = append(params, b.Spec.Params...)
	}
	for _, cb := range clusterbindings {
		params = append(params, cb.Spec.Params...)
	}
	seen := make(map[string]bool, len(params))
	for _, p := range params {
		if seen[p.Name] {
			return nil, fmt.Errorf("duplicate param name: %s", p.Name)
		}
		seen[p.Name] = true
	}
	return params, nil
}
