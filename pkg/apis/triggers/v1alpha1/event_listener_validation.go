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

	"k8s.io/apimachinery/pkg/util/validation"
	"knative.dev/pkg/apis"
)

// Validate EventListener.
func (e *EventListener) Validate(ctx context.Context) *apis.FieldError {
	return e.Spec.validate(ctx)
}

func (s *EventListenerSpec) validate(ctx context.Context) *apis.FieldError {
	if s.Replicas != nil {
		if *s.Replicas < 0 {
			return apis.ErrInvalidValue(*s.Replicas, "spec.replicas")
		}
	}
	if len(s.Triggers) == 0 {
		return apis.ErrMissingField("spec.triggers")
	}
	for i, trigger := range s.Triggers {
		if err := trigger.validate(ctx).ViaField(fmt.Sprintf("spec.triggers[%d]", i)); err != nil {
			return err
		}
	}
	return nil
}

func (t *EventListenerTrigger) validate(ctx context.Context) *apis.FieldError {
	if t.Template == nil && t.TriggerRef == "" {
		return apis.ErrMissingOneOf("template", "triggerRef")
	}

	// Validate optional Bindings
	if err := triggerSpecBindingArray(t.Bindings).validate(ctx); err != nil {
		return err
	}
	if t.Template != nil {
		// Validate required TriggerTemplate
		if err := t.Template.validate(ctx); err != nil {
			return err
		}
	}

	// Validate optional Interceptors
	for i, interceptor := range t.Interceptors {
		if err := interceptor.validate(ctx).ViaField(fmt.Sprintf("interceptors[%d]", i)); err != nil {
			return err
		}
	}

	// The trigger name is added as a label value for 'tekton.dev/trigger' so it must follow the k8s label guidelines:
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
	if errs := validation.IsValidLabelValue(t.Name); len(errs) > 0 {
		return apis.ErrInvalidValue(fmt.Sprintf("trigger name '%s' must be a valid label value", t.Name), "name")
	}

	return nil
}
