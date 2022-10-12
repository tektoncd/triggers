/*
Copyright 2022 The Tekton Authors

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

package v1alpha1_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestInterceptorValidate_OnDelete(t *testing.T) {
	ci := triggersv1.Interceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "github",
		},
		Spec: triggersv1.InterceptorSpec{
			ClientConfig: triggersv1.ClientConfig{
				Service: &triggersv1.ServiceReference{
					Namespace: "",
					Name:      "github-svc",
				},
			},
		},
	}

	err := ci.Validate(apis.WithinDelete(context.Background()))
	if err != nil {
		t.Errorf("Interceptor.Validate() on Delete expected no error, but got one, Interceptor: %v, error: %v", ci, err)
	}
}

func TestInterceptorValidate(t *testing.T) {
	tests := []struct {
		name                  string
		namespacedInterceptor triggersv1.Interceptor
		want                  *apis.FieldError
	}{{
		name: "both URL and Service specified",
		namespacedInterceptor: triggersv1.Interceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "github",
			},
			Spec: triggersv1.InterceptorSpec{
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
		namespacedInterceptor: triggersv1.Interceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "github",
			},
			Spec: triggersv1.InterceptorSpec{
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
		namespacedInterceptor: triggersv1.Interceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "github",
			},
			Spec: triggersv1.InterceptorSpec{
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
			got := tc.namespacedInterceptor.Validate(context.Background())
			if diff := cmp.Diff(tc.want.Error(), got.Error()); diff != "" {
				t.Fatalf("Interceptor.Validate() error: %s", diff)
			}
		})
	}
}
