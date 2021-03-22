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
	"fmt"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/ptr"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

const (
	// OldEscapeAnnotation is used to determine whether or not a TriggerTemplate
	// should retain the old "replace quotes with backslack quote" behaviour
	// when templating in params.
	//
	// This can be removed when this functionality is no-longer needed.
	OldEscapeAnnotation = "triggers.tekton.dev/old-escape-quotes"
)

// Resolve completes the end to end process of taking an event after it has completed processing
// It completes binding the triggerbinding and triggertemplate ref to resolved names and building
// the templated resource to submit after binding params to values in the template
func Resolve(trigger triggersv1.Trigger, getTB getTriggerBinding, getCTB getClusterTriggerBinding, getTT getTriggerTemplate, body []byte, header http.Header, extensions map[string]interface{}) ([]json.RawMessage, error) {

	event, err := newEvent(body, header, extensions)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal event: %w", err)
	}

	trigger, err = resolveResourceNames(trigger, event)
	if err != nil {
		return nil, fmt.Errorf("Failed to resolve resource refs for trigger %s: %w", trigger.GetName(), err)
	}

	rt, err := ResolveTrigger(trigger, getTB, getCTB, getTT)
	if err != nil {
		return nil, fmt.Errorf("Failed to resolve trigger %s: %w", trigger.GetName(), err)
	}

	params, err := resolveParams(rt, event)
	if err != nil {
		return nil, fmt.Errorf("Failed to resolve parameters for trigger %s: %w", trigger.GetName(), err)
	}

	return ResolveResources(rt.TriggerTemplate, params), nil
}

// ResolveParams takes given triggerbindings and produces the resulting
// resource params.
func ResolveParams(rt ResolvedTrigger, body []byte, header http.Header, extensions map[string]interface{}) ([]triggersv1.Param, error) {
	event, err := newEvent(body, header, extensions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}
	return resolveParams(rt, event)
}

func resolveParams(rt ResolvedTrigger, event *event) ([]triggersv1.Param, error) {

	var ttParams []triggersv1.ParamSpec
	if rt.TriggerTemplate != nil {
		ttParams = rt.TriggerTemplate.Spec.Params
	}

	out, err := applyEventValuesToParams(rt.BindingParams, event, ttParams)
	if err != nil {
		return nil, fmt.Errorf("failed to ApplyEventValuesToParams: %w", err)
	}

	return out, nil
}

func resolveResourceNames(trigger triggersv1.Trigger, event *event) (triggersv1.Trigger, error) {
	if trigger.Spec.Bindings != nil {
		var targetBindings = make([]*triggersv1.TriggerSpecBinding, len(trigger.Spec.Bindings))
		for i, binding := range trigger.Spec.Bindings {
			if binding.DynamicRef != "" {
				tb := &triggersv1.TriggerSpecBinding{Ref: binding.Ref}
				resolvedBindingRef, err := resolveExpressionVal(binding.DynamicRef, event)
				if err != nil {
					if binding.Ref == "" {
						return trigger, fmt.Errorf("Failed to resolve triggerbinding expression %s, no fallback provided: %w", binding.DynamicRef, err)
					}
				}
				tb.DynamicRef = resolvedBindingRef
				if string(binding.Kind) != "" {
					tb.Kind = binding.Kind
				}
				if string(binding.APIVersion) != "" {
					tb.APIVersion = binding.APIVersion
				}
				targetBindings[i] = tb
			} else {
				targetBindings[i] = trigger.Spec.Bindings[i]
			}
		}
		trigger.Spec.Bindings = targetBindings
	}
	if trigger.Spec.Template.DynamicRef != nil && *trigger.Spec.Template.DynamicRef != "" {
		resolvedTriggerRef, err := resolveExpressionVal(*trigger.Spec.Template.DynamicRef, event)
		if err != nil {
			if trigger.Spec.Template.Ref == nil || *trigger.Spec.Template.Ref == "" {
				return trigger, fmt.Errorf("Failed to resolve triggertemplate expression %s: %w", *trigger.Spec.Template.Ref, err)
			}
			trigger.Spec.Template.DynamicRef = nil
		} else {
			trigger.Spec.Template.DynamicRef = ptr.String(resolvedTriggerRef)
		}
	}
	return trigger, nil
}

// ResolveResources resolves a templated resource by replacing params with their values.
func ResolveResources(template *triggersv1.TriggerTemplate, params []triggersv1.Param) []json.RawMessage {
	resources := make([]json.RawMessage, len(template.Spec.ResourceTemplates))
	uid := UUID()

	oldEscape := metav1.HasAnnotation(template.ObjectMeta, OldEscapeAnnotation)

	for i := range template.Spec.ResourceTemplates {
		resources[i] = applyParamsToResourceTemplate(params, template.Spec.ResourceTemplates[i].RawExtension.Raw, oldEscape)
		resources[i] = applyUIDToResourceTemplate(resources[i], uid)
	}
	return resources
}

// event represents a HTTP event that Triggers processes
type event struct {
	Header     map[string]string      `json:"header"`
	Body       interface{}            `json:"body"`
	Extensions map[string]interface{} `json:"extensions"`
}

// newEvent returns a new Event from HTTP headers and body
func newEvent(body []byte, headers http.Header, extensions map[string]interface{}) (*event, error) {
	var data interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request body: %w", err)
		}
	}
	joinedHeaders := make(map[string]string, len(headers))
	for k, v := range headers {
		joinedHeaders[k] = strings.Join(v, ",")
	}

	return &event{
		Header:     joinedHeaders,
		Body:       data,
		Extensions: extensions,
	}, nil
}

// applyEventValuesToParams returns a slice of Params with the JSONPath variables replaced
// with values from the event.
func applyEventValuesToParams(params []triggersv1.Param, event *event,
	defaults []triggersv1.ParamSpec) ([]triggersv1.Param, error) {

	allParamsMap := map[string]string{}
	for _, paramSpec := range defaults {
		if paramSpec.Default != nil {
			allParamsMap[paramSpec.Name] = *paramSpec.Default
		}
	}

	for _, p := range params {
		pValue := p.Value
		result, err := resolveExpressionVal(pValue, event)
		if defaults != nil && err != nil {
			// if the header or body was not supplied or was malformed, go with a default if it exists
			v, ok := allParamsMap[p.Name]
			if ok {
				result = v
				err = nil
			}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to replace JSONPath value for param %s: %s: %w", p.Name, p.Value, err)
		}
		allParamsMap[p.Name] = result
	}
	return convertParamMapToArray(allParamsMap), nil
}

func resolveExpressionVal(value string, event *event) (string, error) {
	// Find all expressions wrapped in $() from the value
	expressions, originals := findTektonExpressions(value)
	for i, expr := range expressions {
		val, err := parseJSONPath(event, expr)
		if err != nil {
			return "", fmt.Errorf("failed to replace JSONPath value for expression %s: %w", expr, err)
		}
		value = strings.ReplaceAll(value, originals[i], val)
	}
	return value, nil
}
