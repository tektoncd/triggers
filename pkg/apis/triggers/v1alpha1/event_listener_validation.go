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
	"net/http"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"knative.dev/pkg/apis"
)

var (
	triggerBindings  = SchemeGroupVersion.WithResource("triggerbindings")
	triggerTemplates = SchemeGroupVersion.WithResource("triggertemplates")
	services         = corev1.SchemeGroupVersion.WithResource("services")
)

// Validate EventListener.
func (e *EventListener) Validate(ctx context.Context) *apis.FieldError {
	return e.Spec.validate(ctx, e)
}

func (s *EventListenerSpec) validate(ctx context.Context, el *EventListener) *apis.FieldError {
	if len(s.Triggers) == 0 {
		return apis.ErrMissingField("spec.triggers")
	}
	clientset := ctx.Value("clientSet").(dynamic.Interface)

	for i, t := range s.Triggers {
		// Validate optional TriggerBinding
		if t.Binding != nil {
			if len(t.Binding.Name) == 0 {
				return apis.ErrMissingField(fmt.Sprintf("spec.triggers[%d].binding.name", i))
			}
			_, err := clientset.Resource(triggerBindings).Namespace(el.Namespace).Get(t.Binding.Name, metav1.GetOptions{})
			if err != nil {
				return apis.ErrInvalidValue(err, fmt.Sprintf("spec.triggers[%d].binding.name", i))
			}
		}
		// Validate required TriggerTemplate
		// Optional explicit match
		if len(t.Template.APIVersion) != 0 {
			if t.Template.APIVersion != "v1alpha1" {
				return apis.ErrInvalidValue(fmt.Errorf("Invalid apiVersion"), fmt.Sprintf("spec.triggers[%d].template.apiVersion", i))
			}
		}
		if len(t.Template.Name) == 0 {
			return apis.ErrMissingField(fmt.Sprintf("spec.triggers[%d].template.name", i))
		}
		_, err := clientset.Resource(triggerTemplates).Namespace(el.Namespace).Get(t.Template.Name, metav1.GetOptions{})
		if err != nil {
			return apis.ErrInvalidValue(err, "spec.triggers.template.name")
		}
		if t.Interceptor != nil {
			err := t.Interceptor.validate(ctx, el.Namespace).ViaField(fmt.Sprintf("spec.triggers[%d]", i))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *EventInterceptor) validate(ctx context.Context, namespace string) *apis.FieldError {
	if i.ObjectRef == nil || len(i.ObjectRef.Name) == 0 {
		return apis.ErrMissingField("interceptor.objectRef")
	}
	// Optional explicit match
	if len(i.ObjectRef.Kind) != 0 {
		if i.ObjectRef.Kind != "Service" {
			return apis.ErrInvalidValue(fmt.Errorf("Invalid kind"), "interceptor.objectRef.kind")
		}
	}
	// Optional explicit match
	if len(i.ObjectRef.APIVersion) != 0 {
		if i.ObjectRef.APIVersion != "v1" {
			return apis.ErrInvalidValue(fmt.Errorf("Invalid apiVersion"), "interceptor.objectRef.apiVersion")
		}
	}
	if len(i.ObjectRef.Namespace) != 0 {
		namespace = i.ObjectRef.Namespace
	}
	clientset := ctx.Value("clientSet").(dynamic.Interface)
	_, err := clientset.Resource(services).Namespace(namespace).Get(i.ObjectRef.Name, metav1.GetOptions{})
	if err != nil {
		return apis.ErrInvalidValue(err, "interceptor.objectRef.name")
	}
	for i, header := range i.Header {
		// Enforce non-empty canonical header keys
		if len(header.Name) == 0 || http.CanonicalHeaderKey(header.Name) != header.Name {
			return apis.ErrInvalidValue(fmt.Errorf("Invalid header name"), fmt.Sprintf("interceptor.header[%d].name", i))
		}
		// Enforce non-empty header values
		if header.Value.Type == pipelinev1.ParamTypeString {
			if len(header.Value.StringVal) == 0 {
				return apis.ErrInvalidValue(fmt.Errorf("Invalid header value"), fmt.Sprintf("interceptor.header[%d].value", i))
			}
		} else if len(header.Value.ArrayVal) == 0 {
			return apis.ErrInvalidValue(fmt.Errorf("Invalid header value"), fmt.Sprintf("interceptor.header[%d].value", i))
		}
	}
	return nil
}
