package v1alpha1_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestResolveAddress(t *testing.T) {
	tests := []struct {
		name string
		it   *v1alpha1.InterceptorType
		want string
	}{{
		name: "clientConfig.url is specified",
		it: &v1alpha1.InterceptorType{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: v1alpha1.InterceptorTypeSpec{
				ClientConfig: v1alpha1.ClientConfig{
					URL: &apis.URL{
						Scheme: "http",
						Host:   "foo.bar.com:8081",
						Path:   "abc",
					},
				},
			},
		},
		want: "http://foo.bar.com:8081/abc",
	}, {
		name: "clientConfig.service with namespace",
		it: &v1alpha1.InterceptorType{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: v1alpha1.InterceptorTypeSpec{
				ClientConfig: v1alpha1.ClientConfig{
					Service: &v1alpha1.ServiceReference{
						Name:      "my-svc",
						Namespace: "default",
						Path:      "blah",
						Port:      8081,
					},
				},
			},
		},
		want: "http://my-svc.default.svc:8081/blah",
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.it.ResolveAddress()
			if err != nil {
				t.Fatalf("ResolveAddress() unpexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got.String()); diff != "" {
				t.Fatalf("ResolveAddress -want/+got: %s", diff)
			}
		})
	}

	t.Run("clientConfig with nil url", func(t *testing.T) {
		it := &v1alpha1.InterceptorType{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: v1alpha1.InterceptorTypeSpec{
				ClientConfig: v1alpha1.ClientConfig{
					URL:     nil,
					Service: nil,
				},
			},
		}
		_, err := it.ResolveAddress()
		if !errors.Is(err, v1alpha1.ErrNilURL) {
			t.Fatalf("ResolveToURL expected error to be %s but got %s", v1alpha1.ErrNilURL, err)
		}
	})
}
