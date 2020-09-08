/*
Copyright 2020 The Tekton Authors

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

	"github.com/google/cel-go/cel"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/validate"
	"knative.dev/pkg/apis"
)

// Validate validates a Trigger
func (t *Trigger) Validate(ctx context.Context) *apis.FieldError {
	if err := validate.ObjectMetadata(t.GetObjectMeta()); err != nil {
		return err.ViaField("metadata")
	}
	return t.Spec.validate(ctx).ViaField("spec")
}

func (t *TriggerSpec) validate(ctx context.Context) *apis.FieldError {
	// Validate optional Bindings
	if err := triggerSpecBindingArray(t.Bindings).validate(ctx); err != nil {
		return err
	}
	// Validate required TriggerTemplate
	if err := t.Template.validate(ctx); err != nil {
		return err
	}

	// Validate optional Interceptors
	for i, interceptor := range t.Interceptors {
		if err := interceptor.validate(ctx).ViaField(fmt.Sprintf("interceptors[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

func (t TriggerSpecTemplate) validate(ctx context.Context) *apis.FieldError {
	// Optional explicit match
	if t.APIVersion != "" {
		if t.APIVersion != "v1alpha1" {
			return apis.ErrInvalidValue(fmt.Errorf("invalid apiVersion"), "template.apiVersion")
		}
	}
	if t.Name == "" {
		return apis.ErrMissingField("template.name")
	}
	return nil

}

func (t triggerSpecBindingArray) validate(ctx context.Context) *apis.FieldError {
	if len(t) > 0 {
		for i, b := range t {
			// Either Ref or Spec should be present
			if b.Ref == "" && b.Spec == nil {
				return apis.ErrMissingOneOf(fmt.Sprintf("bindings[%d].Ref", i), fmt.Sprintf("bindings[%d].Spec", i))
			}

			// Both Ref and Spec can't be present at the same time
			if b.Ref != "" && b.Spec != nil {
				return apis.ErrMultipleOneOf(fmt.Sprintf("bindings[%d].Ref", i), fmt.Sprintf("bindings[%d].Spec", i))
			}

			if b.Ref != "" && b.Kind != NamespacedTriggerBindingKind && b.Kind != ClusterTriggerBindingKind {
				return apis.ErrInvalidValue(fmt.Errorf("invalid kind"), fmt.Sprintf("bindings[%d].kind", i))
			}
		}
	}
	return nil
}

func (i *TriggerInterceptor) validate(ctx context.Context) *apis.FieldError {
	if i.Webhook == nil && i.GitHub == nil && i.GitLab == nil && i.CEL == nil && i.Bitbucket == nil {
		return apis.ErrMissingField("interceptor")
	}

	// Enforce oneof
	numSet := 0
	if i.Webhook != nil {
		numSet++
	}
	if i.GitHub != nil {
		numSet++
	}
	if i.GitLab != nil {
		numSet++
	}
	if i.Bitbucket != nil {
		numSet++
	}

	if numSet > 1 {
		return apis.ErrMultipleOneOf("interceptor.webhook", "interceptor.github", "interceptor.gitlab")
	}

	if i.Webhook != nil {
		if i.Webhook.ObjectRef == nil || i.Webhook.ObjectRef.Name == "" {
			return apis.ErrMissingField("interceptor.webhook.objectRef")
		}
		w := i.Webhook
		if w.ObjectRef.Kind != "Service" {
			return apis.ErrInvalidValue(fmt.Errorf("invalid kind"), "interceptor.webhook.objectRef.kind")
		}

		// Optional explicit match
		if w.ObjectRef.APIVersion != "v1" {
			return apis.ErrInvalidValue(fmt.Errorf("invalid apiVersion"), "interceptor.webhook.objectRef.apiVersion")
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
	// if i.GitHub != nil {
	//
	// }

	// No gitlab validation required yet.
	// if i.GitLab != nil {
	//
	// }

	if i.CEL != nil {
		if i.CEL.Filter == "" && len(i.CEL.Overlays) == 0 {
			return apis.ErrMultipleOneOf("cel.filter", "cel.overlays")
		}
		env, err := cel.NewEnv()
		if err != nil {
			return apis.ErrInvalidValue(fmt.Errorf("failed to create a CEL env: %s", err), "cel.filter")
		}
		if i.CEL.Filter != "" {
			if _, issues := env.Parse(i.CEL.Filter); issues != nil && issues.Err() != nil {
				return apis.ErrInvalidValue(fmt.Errorf("failed to parse the CEL filter: %s", issues.Err()), "cel.filter")
			}
		}
		for _, v := range i.CEL.Overlays {
			if _, issues := env.Parse(v.Expression); issues != nil && issues.Err() != nil {
				return apis.ErrInvalidValue(fmt.Errorf("failed to parse the CEL overlay: %s", issues.Err()), "cel.overlay")
			}
		}
	}
	return nil
}
