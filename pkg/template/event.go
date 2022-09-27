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

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
)

const (
	// OldEscapeAnnotation is used to determine whether or not a TriggerTemplate
	// should retain the old "replace quotes with backslack quote" behaviour
	// when templating in params.
	//
	// This can be removed when this functionality is no-longer needed.
	OldEscapeAnnotation = "triggers.tekton.dev/old-escape-quotes"
)

type TriggerContext struct {
	EventID string `json:"eventID"`
}

func NewTriggerContext(eventID string) TriggerContext {
	return TriggerContext{EventID: eventID}
}

// ResolveParams takes given triggerbindings and produces the resulting
// resource params.
func ResolveParams(rt ResolvedTrigger, body []byte, header http.Header, extensions map[string]interface{}, triggerContext TriggerContext) ([]triggersv1.Param, error) {
	var ttParams []triggersv1.ParamSpec
	if rt.TriggerTemplate != nil {
		ttParams = rt.TriggerTemplate.Spec.Params
	}

	out, err := applyEventValuesToParams(rt.BindingParams, body, header, extensions, ttParams, triggerContext)
	if err != nil {
		return nil, fmt.Errorf("failed to ApplyEventValuesToParams: %w", err)
	}

	return out, nil
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
	Context    TriggerContext         `json:"context"`
}

// newEvent returns a new Event from HTTP headers and body
func newEvent(body []byte, headers http.Header, extensions map[string]interface{}, triggerContext TriggerContext) (*event, error) {
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
		Context:    triggerContext,
	}, nil
}

// applyEventValuesToParams returns a slice of Params with the JSONPath variables replaced
// with values from the event body, headers, and extensions.
func applyEventValuesToParams(params []triggersv1.Param, body []byte, header http.Header, extensions map[string]interface{},
	defaults []triggersv1.ParamSpec,
	triggerContext TriggerContext) ([]triggersv1.Param, error) {
	event, err := newEvent(body, header, extensions, triggerContext)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	allParamsMap := map[string]string{}
	for _, paramSpec := range defaults {
		if paramSpec.Default != nil {
			allParamsMap[paramSpec.Name] = *paramSpec.Default
		}
	}

	for _, p := range params {
		pValue := p.Value
		// Find all expressions wrapped in $() from the value
		expressions, originals := findTektonExpressions(pValue)
		for i, expr := range expressions {
			val, err := parseJSONPath(event, expr)
			if defaults != nil && err != nil {
				// if the header or body was not supplied or was malformed, go with a default if it exists
				v, ok := allParamsMap[p.Name]
				if ok {
					val = v
					err = nil
				}
			}
			if err != nil {
				return nil, fmt.Errorf("failed to replace JSONPath value for param %s: %s: %w", p.Name, p.Value, err)
			}
			pValue = strings.ReplaceAll(pValue, originals[i], val)
		}
		allParamsMap[p.Name] = pValue
	}
	return convertParamMapToArray(allParamsMap), nil
}
