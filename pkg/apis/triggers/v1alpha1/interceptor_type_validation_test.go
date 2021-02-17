package v1alpha1_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestInterceptorTypeValidate(t *testing.T) {
	tests := []struct {
		name            string
		interceptorType triggersv1.InterceptorType
		want            *apis.FieldError
	}{{
		name: "both URL and Service specified",
		interceptorType: triggersv1.InterceptorType{
			ObjectMeta: metav1.ObjectMeta{
				Name: "github",
			},
			Spec: triggersv1.InterceptorTypeSpec{
				ClientConfig: triggersv1.ClientConfig{
					URL: &apis.URL{
						Scheme: "http",
						Host:   "some.host",
					},
					Service: &triggersv1.ServiceReference{
						Name:      "github-svc",
						Namespace: "default",
					},
				},
			},
		},
		want: apis.ErrMultipleOneOf("spec.clientConfig.url", "spec.clientConfig.service"),
	}, {
		name: "service missing namespace",
		interceptorType: triggersv1.InterceptorType{
			ObjectMeta: metav1.ObjectMeta{
				Name: "github",
			},
			Spec: triggersv1.InterceptorTypeSpec{
				ClientConfig: triggersv1.ClientConfig{
					Service: &triggersv1.ServiceReference{
						Namespace: "",
						Name:      "github-svc",
					},
				},
			},
		},
		want: apis.ErrMissingField("spec.clientConfig.service.namespace"),
	}, {
		name: "service missing name",
		interceptorType: triggersv1.InterceptorType{
			ObjectMeta: metav1.ObjectMeta{
				Name: "github",
			},
			Spec: triggersv1.InterceptorTypeSpec{
				ClientConfig: triggersv1.ClientConfig{
					Service: &triggersv1.ServiceReference{
						Namespace: "default",
						Name:      "",
					},
				},
			},
		},
		want: apis.ErrMissingField("spec.clientConfig.service.name"),
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.interceptorType.Validate(context.Background())
			if diff := cmp.Diff(tc.want.Error(), got.Error()); diff != "" {
				t.Fatalf("InterceptorType.Validate() error: %s", diff)
			}
		})
	}
}
