package v1alpha1

import (
	"context"
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/validate"
	"knative.dev/pkg/apis"
)

// Validate performs validation on Trigger fields
func (tr *Trigger) Validate(ctx context.Context) *apis.FieldError {
	if err := validate.ObjectMetadata(tr.GetObjectMeta()); err != nil {
		return err.ViaField("metadata")
	}
	return tr.Spec.validate(ctx)
}

func (t *TriggerSpec) validate(ctx context.Context) *apis.FieldError {

	// Validate optional Bindings
	for i, b := range t.Bindings {
		if b.Name == "" {
			return apis.ErrMissingField(fmt.Sprintf("bindings[%d].name", i))
		}

		if b.Kind != NamespacedTriggerBindingKind && b.Kind != ClusterTriggerBindingKind {
			return apis.ErrInvalidValue(fmt.Errorf("invalid kind"), fmt.Sprintf("bindings[%d].kind", i))
		}
	}
	// Validate required TriggerTemplate
	// Optional explicit match
	if t.Template.APIVersion != "" {
		if t.Template.APIVersion != "v1alpha1" {
			return apis.ErrInvalidValue(fmt.Errorf("invalid apiVersion"), "template.apiVersion")
		}
	}
	if t.Template.Name == "" {
		return apis.ErrMissingField(fmt.Sprintf("template.name"))
	}
	for i, interceptor := range t.Interceptors {
		if err := interceptor.validate(ctx).ViaField(fmt.Sprintf("interceptors[%d]", i)); err != nil {
			return err
		}
	}
	return nil
}
