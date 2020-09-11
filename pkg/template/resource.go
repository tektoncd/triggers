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
	"context"
	"encoding/json"
	"fmt"
	"strings"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/rand"
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

type getTriggerBinding func(ctx context.Context, name string, options metav1.GetOptions) (*triggersv1.TriggerBinding, error)
type getTriggerTemplate func(ctx context.Context, name string, options metav1.GetOptions) (*triggersv1.TriggerTemplate, error)
type getClusterTriggerBinding func(ctx context.Context, name string, options metav1.GetOptions) (*triggersv1.ClusterTriggerBinding, error)

// ResolveTrigger takes in a trigger containing object refs to bindings and
// templates and resolves them to their underlying values.
func ResolveTrigger(trigger triggersv1.EventListenerTrigger, getTB getTriggerBinding, getCTB getClusterTriggerBinding, getTT getTriggerTemplate) (ResolvedTrigger, error) {
	bp, err := resolveBindingsToParams(trigger.Bindings, getTB, getCTB)
	if err != nil {
		return ResolvedTrigger{}, fmt.Errorf("failed to resolve bindings: %w", err)
	}
	tt, err := getTT(context.Background(), trigger.Template.Name, metav1.GetOptions{})
	if err != nil {
		return ResolvedTrigger{}, fmt.Errorf("error getting TriggerTemplate %s: %w", trigger.Template.Name, err)
	}
	return ResolvedTrigger{TriggerTemplate: tt, BindingParams: bp}, nil
}

// resolveBindingsToParams takes in both embedded bindings and references and returns a list of resolved Param values.ResolveBindingsToParams
func resolveBindingsToParams(bindings []*triggersv1.TriggerSpecBinding, getTB getTriggerBinding, getCTB getClusterTriggerBinding) ([]triggersv1.Param, error) {
	bindingParams := []triggersv1.Param{}
	for _, b := range bindings {
		switch {
		case b.Spec != nil: // Could also call SetDefaults and not rely on this?
			bindingParams = append(bindingParams, b.Spec.Params...)

		case b.Name != "" && b.Value != nil:
			bindingParams = append(bindingParams, triggersv1.Param{
				Name:  b.Name,
				Value: *b.Value,
			})

		case b.Ref != "" && b.Kind == triggersv1.ClusterTriggerBindingKind:
			ctb, err := getCTB(context.Background(), b.Ref, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("error getting ClusterTriggerBinding %s: %w", b.Name, err)
			}
			bindingParams = append(bindingParams, ctb.Spec.Params...)

		case b.Ref != "": // if no kind is set, assume NamespacedTriggerBinding
			tb, err := getTB(context.Background(), b.Ref, metav1.GetOptions{})
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

// mergeInDefaultParams returns the params with the addition of all
// paramSpecs that have default values and are already in the params list
func mergeInDefaultParams(params []triggersv1.Param, paramSpecs []triggersv1.ParamSpec) []triggersv1.Param {
	allParamsMap := map[string]string{}
	for _, paramSpec := range paramSpecs {
		if paramSpec.Default != nil {
			allParamsMap[paramSpec.Name] = *paramSpec.Default
		}
	}
	for _, param := range params {
		allParamsMap[param.Name] = param.Value
	}
	return convertParamMapToArray(allParamsMap)
}

// applyParamsToResourceTemplate returns the TriggerResourceTemplate with the
// param values substituted for all matching param variables in the template
func applyParamsToResourceTemplate(params []triggersv1.Param, rt json.RawMessage) json.RawMessage {
	// Assume the params are valid
	for _, param := range params {
		rt = applyParamToResourceTemplate(param, rt)
	}
	return rt
}

// applyParamToResourceTemplate returns the TriggerResourceTemplate with the
// param value substituted for all matching param variables in the template
func applyParamToResourceTemplate(param triggersv1.Param, rt json.RawMessage) json.RawMessage {
	// Assume the param is valid
	paramVariable := fmt.Sprintf("$(tt.params.%s)", param.Name)
	// Escape quotes so that that JSON strings can be appended to regular strings.
	// See #257 for discussion on this behavior.
	paramValue := strings.Replace(param.Value, `"`, `\"`, -1)
	rt = bytes.Replace(rt, []byte(paramVariable), []byte(paramValue), -1)
	return rt
}

// UID generates a random string like the Kubernetes apiserver generateName metafield postfix.
var UID = func() string { return rand.String(5) }

// applyUIDToResourceTemplate returns the TriggerResourceTemplate after uid replacement
// The same uid should be used per trigger to properly address resources throughout the TriggerTemplate.
func applyUIDToResourceTemplate(rt json.RawMessage, uid string) json.RawMessage {
	return bytes.Replace(rt, uidMatch, []byte(uid), -1)
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
