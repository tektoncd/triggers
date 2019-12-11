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

// clientKey is used as the key for associating information
// with a context.Context.
type clientKey struct{}

// WithClientSet adds the dynamic clientset to the context.
func WithClientSet(ctx context.Context, i dynamic.Interface) context.Context {
	return context.WithValue(ctx, clientKey{}, i)
}

// Validate EventListener.
func (e *EventListener) Validate(ctx context.Context) *apis.FieldError {
	return e.Spec.validate(ctx, e)
}

func (s *EventListenerSpec) validate(ctx context.Context, el *EventListener) *apis.FieldError {
	if len(s.Triggers) == 0 {
		return apis.ErrMissingField("spec.triggers")
	}
	clientset := ctx.Value(clientKey{}).(dynamic.Interface)
	for i, t := range s.Triggers {
		// Validate that only one of binding or bindings is set
		if t.DeprecatedBinding != nil && len(t.Bindings) > 0 {
			return apis.ErrMultipleOneOf(fmt.Sprintf("spec.triggers[%d].binding", i), fmt.Sprintf("spec.triggers[%d].binding", i))
		}
		// Validate optional TriggerBinding
		for j, b := range t.Bindings {
			if len(b.Name) == 0 {
				return apis.ErrMissingField(fmt.Sprintf("spec.triggers[%d].bindings[%d].name", i, j))
			}
			_, err := clientset.Resource(triggerBindings).Namespace(el.Namespace).Get(b.Name, metav1.GetOptions{})
			if err != nil {
				return apis.ErrInvalidValue(err, fmt.Sprintf("spec.triggers[%d].bindings[%d].name", i, j))
			}
		}
		// Validate required TriggerTemplate
		// Optional explicit match
		if len(t.Template.APIVersion) != 0 {
			if t.Template.APIVersion != "v1alpha1" {
				return apis.ErrInvalidValue(fmt.Errorf("invalid apiVersion"), fmt.Sprintf("spec.triggers[%d].template.apiVersion", i))
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
	// Validate at least one
	if i.Webhook == nil && i.Github == nil && i.Gitlab == nil {
		return apis.ErrMissingField(("interceptor"))
	}

	// Enforce oneof
	numSet := 0
	if i.Webhook != nil {
		numSet++
	}
	if i.Github != nil {
		numSet++
	}
	if i.Gitlab != nil {
		numSet++
	}

	if numSet > 1 {
		return apis.ErrMultipleOneOf("interceptor.webhook", "interceptor.github", "interceptor.gitlab")
	}

	if i.Webhook != nil {
		if i.Webhook.ObjectRef == nil || len(i.Webhook.ObjectRef.Name) == 0 {
			return apis.ErrMissingField("interceptor.webhook")
		}
		w := i.Webhook
		if len(w.ObjectRef.Kind) != 0 {
			if w.ObjectRef.Kind != "Service" {
				return apis.ErrInvalidValue(fmt.Errorf("invalid kind"), "interceptor.webhook.objectRef.kind")
			}
		}
		// Optional explicit match
		if len(w.ObjectRef.APIVersion) != 0 {
			if w.ObjectRef.APIVersion != "v1" {
				return apis.ErrInvalidValue(fmt.Errorf("invalid apiVersion"), "interceptor.webhook.objectRef.apiVersion")
			}
		}
		if len(w.ObjectRef.Namespace) != 0 {
			namespace = w.ObjectRef.Namespace
		}

		clientset := ctx.Value(clientKey{}).(dynamic.Interface)
		_, err := clientset.Resource(services).Namespace(namespace).Get(w.ObjectRef.Name, metav1.GetOptions{})
		if err != nil {
			return apis.ErrInvalidValue(err, "interceptor.webhook.objectRef.name")
		}
		for i, header := range w.Header {
			// Enforce non-empty canonical header keys
			if len(header.Name) == 0 || http.CanonicalHeaderKey(header.Name) != header.Name {
				return apis.ErrInvalidValue(fmt.Errorf("invalid header name"), fmt.Sprintf("interceptor.webhook.header[%d].name", i))
			}
			// Enforce non-empty header values
			if header.Value.Type == pipelinev1.ParamTypeString {
				if len(header.Value.StringVal) == 0 {
					return apis.ErrInvalidValue(fmt.Errorf("invalid header value"), fmt.Sprintf("interceptor.webhook.header[%d].value", i))
				}
			} else if len(header.Value.ArrayVal) == 0 {
				return apis.ErrInvalidValue(fmt.Errorf("invalid header value"), fmt.Sprintf("interceptor.webhook.header[%d].value", i))
			}
		}
	}

	// No github validation required yet.
	// if i.Github != nil {
	//
	// }

	// No gitlab validation required yet.
	// if i.Gitlab != nil {
	//
	// }
	return nil
}
