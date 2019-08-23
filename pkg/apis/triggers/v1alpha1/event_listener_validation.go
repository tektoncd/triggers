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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"knative.dev/pkg/apis"
)

// Validate EventListener.
func (t *EventListener) Validate(ctx context.Context) *apis.FieldError {
	return t.Spec.Validate(ctx, t)
}

// Validate EventListenerSpec.
func (s *EventListenerSpec) Validate(ctx context.Context, el *EventListener) *apis.FieldError {
	if len(s.Triggers) == 0 {
		return apis.ErrMissingField("spec.triggers")
	}

	clientset := ctx.Value("clientSet").(dynamic.Interface)
	TriggerBindings := SchemeGroupVersion.WithResource("triggerbindings")
	TriggerTemplates := SchemeGroupVersion.WithResource("triggertemplates")

	for n, t := range s.Triggers {
		if t.Binding != (EventListenerBinding{}) {
			_, err := clientset.Resource(TriggerBindings).Namespace(el.Namespace).Get(t.Binding.Name, metav1.GetOptions{})
			if err != nil {
				return apis.ErrInvalidValue(err, "spec.triggers.binding")
			}
		}
		if t.Template == (EventListenerTemplate{}) {
			return apis.ErrMissingField("spec.triggers.template")
		} else {
			_, err := clientset.Resource(TriggerTemplates).Namespace(el.Namespace).Get(t.Template.Name, metav1.GetOptions{})
			if err != nil {
				return apis.ErrInvalidValue(err, "spec.triggers.template")
			}
		}
		if t.Interceptor != nil {
			return t.Interceptor.Validate(ctx).ViaField(fmt.Sprintf("spec.triggers[%d]", n))
		}
	}
	return nil
}

func (i *EventInterceptor) Validate(ctx context.Context) *apis.FieldError {
	if i.ObjectRef == nil {
		return apis.ErrMissingField("objectRef")
	}
	return nil
}
