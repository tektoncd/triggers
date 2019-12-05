package v1alpha1_test

import (
	"context"
	"testing"

	v1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	bldr "github.com/tektoncd/triggers/test/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamicclient "k8s.io/client-go/dynamic/fake"
)

func Test_EventListenerValidate_error(t *testing.T) {
	tests := []struct {
		name     string
		el       *v1alpha1.EventListener
		raiseErr bool
	}{{
		name: "TriggerTemplate Does Not Exist",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tb", "dne", "v1alpha1"))),
		raiseErr: true,
	}, {
		name: "Interceptor Does Not Exist",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tb", "tt", "v1alpha1",
					bldr.EventListenerTriggerInterceptor("dne", "v1", "Service", "")))),
		raiseErr: true,
	}, {
		name: "Interceptor Missing ObjectRef",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings:    []*v1alpha1.EventListenerBinding{{Name: "tb"}},
					Template:    v1alpha1.EventListenerTemplate{Name: "tt"},
					Interceptor: &v1alpha1.EventInterceptor{},
				}},
			},
		},
		raiseErr: true,
	}, {
		name: "Interceptor Wrong APIVersion",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tb", "tt", "v1alpha1",
					bldr.EventListenerTriggerInterceptor("foo", "v3", "Service", "")))),
		raiseErr: true,
	}, {
		name: "Interceptor Wrong Kind",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tb", "tt", "v1alpha1",
					bldr.EventListenerTriggerInterceptor("foo", "v1", "Deployment", "")))),
		raiseErr: true,
	}, {
		name: "Interceptor Non-Canonical Header",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tb", "tt", "v1alpha1",
					bldr.EventListenerTriggerInterceptor("foo", "v1", "Deployment", "",
						bldr.EventInterceptorParam("non-canonical-header-key", "valid value"))))),
		raiseErr: true,
	}, {
		name: "Interceptor Empty Header Name",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tb", "tt", "v1alpha1",
					bldr.EventListenerTriggerInterceptor("foo", "v1", "Deployment", "",
						bldr.EventInterceptorParam("", "valid value"))))),
		raiseErr: true,
	}, {
		name: "Interceptor Empty Header Value",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tb", "tt", "v1alpha1",
					bldr.EventListenerTriggerInterceptor("foo", "v1", "Deployment", "",
						bldr.EventInterceptorParam("Valid-Header-Key", ""))))),
		raiseErr: true,
	}, {
		name: "Valid EventListener No TriggerBinding",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("", "tt", "v1alpha1"))),
		raiseErr: false,
	}, {
		name: "Valid EventListener No Interceptor",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tb", "tt", "v1alpha1"))),
		raiseErr: false,
	}, {
		name: "Valid EventListener Interceptor Name only",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tb", "tt", "v1alpha1",
					bldr.EventListenerTriggerInterceptor("svc", "", "", "")))),
		raiseErr: false,
	}, {
		name: "Valid EventListener Interceptor",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tb", "tt", "v1alpha1",
					bldr.EventListenerTriggerInterceptor("svc", "v1", "Service", "namespace")))),
		raiseErr: false,
	}, {
		name: "Valid EventListener Interceptor With Header",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tb", "tt", "v1alpha1",
					bldr.EventListenerTriggerInterceptor("svc", "v1", "Service", "namespace",
						bldr.EventInterceptorParam("Valid-Header-Key", "valid value"))))),
		raiseErr: false,
	}, {
		name: "Valid EventListener Interceptor With Headers",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tb", "tt", "v1alpha1",
					bldr.EventListenerTriggerInterceptor("svc", "v1", "Service", "namespace",
						bldr.EventInterceptorParam("Valid-Header-Key1", "valid value1"),
						bldr.EventInterceptorParam("Valid-Header-Key1", "valid value2"),
						bldr.EventInterceptorParam("Valid-Header-Key2", "valid value"))))),
		raiseErr: false,
	}, {
		name: "Valid EventListener Two Triggers",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tb", "tt", "v1alpha1",
					bldr.EventListenerTriggerInterceptor("svc", "v1", "Service", "namespace"),
				),
				bldr.EventListenerTrigger("tb", "tt", "v1alpha1"))),
		raiseErr: false,
	},
	}

	tb := bldr.TriggerBinding("tb", "namespace",
		bldr.TriggerBindingSpec(
			bldr.TriggerBindingParam("oneparam", "$(event.one)"),
			bldr.TriggerBindingParam("twoparamname", "$(event.two.name)"),
		),
	)

	tt := bldr.TriggerTemplate("tt", "namespace",
		bldr.TriggerTemplateSpec(
			bldr.TriggerTemplateParam("foo", "desc", "val"),
			bldr.TriggerResourceTemplate(simpleResourceTemplate)))

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	triggerBinding := v1alpha1.SchemeGroupVersion.WithKind("TriggerBinding")
	scheme.AddKnownTypeWithName(triggerBinding,
		&v1alpha1.TriggerBinding{},
	)

	triggerTemplate := v1alpha1.SchemeGroupVersion.WithKind("TriggerTemplate")
	scheme.AddKnownTypeWithName(triggerTemplate,
		&v1alpha1.TriggerTemplate{},
	)

	service := corev1.SchemeGroupVersion.WithKind("Service")
	scheme.AddKnownTypeWithName(service,
		&corev1.Service{},
	)

	dynamicClient := fakedynamicclient.NewSimpleDynamicClient(scheme, tb, tt, svc)
	ctx := v1alpha1.WithClientSet(context.TODO(), dynamicClient)
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
