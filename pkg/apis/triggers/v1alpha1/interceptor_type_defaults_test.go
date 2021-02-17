package v1alpha1_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInterceptorTypeSetDefaults(t *testing.T) {
	tests := []struct {
		name string
		in   triggersv1.InterceptorType
		want triggersv1.InterceptorType
	}{{
		name: "sets default service port",
		in: triggersv1.InterceptorType{
			ObjectMeta: metav1.ObjectMeta{
				Name: "github",
			},
			Spec: triggersv1.InterceptorTypeSpec{
				ClientConfig: triggersv1.ClientConfig{
					Service: &triggersv1.ServiceReference{
						Namespace: "default",
						Name:      "github-svc",
					},
				},
			},
		},
		want: triggersv1.InterceptorType{
			ObjectMeta: metav1.ObjectMeta{
				Name: "github",
			},
			Spec: triggersv1.InterceptorTypeSpec{
				ClientConfig: triggersv1.ClientConfig{
					Service: &triggersv1.ServiceReference{
						Namespace: "default",
						Name:      "github-svc",
						Port:      80,
					},
				},
			},
		},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in
			got.SetDefaults(triggersv1.WithUpgradeViaDefaulting(context.Background()))
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("InterceptorType SetDefaults error: %s", diff)
			}
		})
	}
}
