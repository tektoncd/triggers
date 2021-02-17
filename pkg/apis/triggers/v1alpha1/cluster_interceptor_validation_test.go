package v1alpha1_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestClusterInterceptorValidate(t *testing.T) {
	tests := []struct {
		name               string
		clusterInterceptor triggersv1.ClusterInterceptor
		want               *apis.FieldError
	}{{
		name: "both URL and Service specified",
		clusterInterceptor: triggersv1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "github",
			},
			Spec: triggersv1.ClusterInterceptorSpec{
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
		clusterInterceptor: triggersv1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "github",
			},
			Spec: triggersv1.ClusterInterceptorSpec{
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
		clusterInterceptor: triggersv1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "github",
			},
			Spec: triggersv1.ClusterInterceptorSpec{
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
			got := tc.clusterInterceptor.Validate(context.Background())
			if diff := cmp.Diff(tc.want.Error(), got.Error()); diff != "" {
				t.Fatalf("ClusterInterceptor.Validate() error: %s", diff)
			}
		})
	}
}
