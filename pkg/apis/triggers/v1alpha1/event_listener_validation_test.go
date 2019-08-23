package v1alpha1_test

import (
	"context"
	"testing"

	v1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	bldr "github.com/tektoncd/triggers/test/builder"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamicclient "k8s.io/client-go/dynamic/fake"
)

func Test_EventListenerValidate_error(t *testing.T) {
	tests := []struct {
		name     string
		el       *v1alpha1.EventListener
		raiseErr bool
	}{
		{
			name: "without correct triggerBinding",
			el: bldr.EventListener("name", "namespace",
				bldr.EventListenerSpec(
					bldr.EventListenerTrigger("tb2", "tt1", "v1alpha1"))),
			raiseErr: true,
		},
		{
			name: "without correct triggerTemplate",
			el: bldr.EventListener("name", "namespace",
				bldr.EventListenerSpec(
					bldr.EventListenerTrigger("tb1", "tt2", "v1alpha1"))),
			raiseErr: true,
		},
		{
			name: "validate",
			el: bldr.EventListener("name", "namespace",
				bldr.EventListenerSpec(
					bldr.EventListenerTrigger("tb1", "tt1", "v1alpha1"))),
			raiseErr: false,
		},
	}

	tb := bldr.TriggerBinding("tb1", "namespace",
		bldr.TriggerBindingSpec(
			bldr.TriggerBindingParam("oneparam", "$(event.one)"),
			bldr.TriggerBindingParam("twoparamname", "$(event.two.name)"),
		),
	)

	tt := bldr.TriggerTemplate("tt1", "namespace",
		bldr.TriggerTemplateSpec(
			bldr.TriggerTemplateParam("foo", "desc", "val"),
			bldr.TriggerResourceTemplate(simpleResourceTemplate)))

	scheme := runtime.NewScheme()
	TriggerBinding := v1alpha1.SchemeGroupVersion.WithKind("TriggerBinding")
	scheme.AddKnownTypeWithName(TriggerBinding,
		&v1alpha1.TriggerBinding{},
	)

	TriggerTemplate := v1alpha1.SchemeGroupVersion.WithKind("TriggerTemplate")
	scheme.AddKnownTypeWithName(TriggerTemplate,
		&v1alpha1.TriggerTemplate{},
	)

	dynamicClient := fakedynamicclient.NewSimpleDynamicClient(scheme, tb, tt)
	ctx := context.TODO()
	ctx = context.WithValue(ctx, "clientSet", dynamicClient)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.el.Validate(ctx)
			if test.raiseErr && err == nil {
				t.Errorf("EventListener.Validate() expected error, but get none, EventListener: %v", test.el)
			} else if !test.raiseErr && err != nil {
				t.Errorf("EventListener.Validate() expected no error, but get one, EventListener: %v, error: %v", test.el, err)
			}
		})
	}
}
