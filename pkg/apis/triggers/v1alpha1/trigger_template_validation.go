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

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	"knative.dev/pkg/apis"
)

// Validate validates a TriggerTemplate.
func (t *TriggerTemplate) Validate(ctx context.Context) *apis.FieldError {
	// TODO: Add metadata validation as in pipeline
	return t.Spec.validate(ctx).ViaField("spec")
}

// Validate validates a TriggerTemplateSpec.
func (s *TriggerTemplateSpec) validate(ctx context.Context) *apis.FieldError {
	if equality.Semantic.DeepEqual(s, TriggerTemplateSpec{}) {
		return apis.ErrMissingField(apis.CurrentField)
	}
	if len(s.ResourceTemplates) == 0 {
		return apis.ErrMissingField("resourcetemplates")
	}
	if err := validateResourceTemplates(s.ResourceTemplates).ViaField("resourcetemplates"); err != nil {
		return err
	}
	return nil
}

func validateResourceTemplates(templates []TriggerResourceTemplate) *apis.FieldError {
	for i, trt := range templates {
		apiVersion, kind := trt.getAPIVersionAndKind()

		if apiVersion == "" {
			return apis.ErrMissingField(fmt.Sprintf("[%d].apiVersion", i))
		}

		if kind == "" {
			return apis.ErrMissingField(fmt.Sprintf("[%d].kind", i))
		}

		if !trt.IsAllowedType() {
			return apis.ErrInvalidValue(
				fmt.Sprintf("resource type not allowed: apiVersion: %s, kind: %s", apiVersion, kind),
				fmt.Sprintf("[%d]", i))
		}
	}
	return nil
}
