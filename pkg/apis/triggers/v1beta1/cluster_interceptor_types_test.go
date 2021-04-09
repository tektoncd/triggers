/*
Copyright 2021 The Tekton Authors

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

package v1beta1_test

import (
	"errors"
	"testing"

	"knative.dev/pkg/ptr"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestResolveAddress(t *testing.T) {
	tests := []struct {
		name string
		it   *v1beta1.ClusterInterceptor
		want string
	}{{
		name: "clientConfig.url is specified",
		it: &v1beta1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: v1beta1.ClusterInterceptorSpec{
				ClientConfig: v1beta1.ClientConfig{
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
		it: &v1beta1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: v1beta1.ClusterInterceptorSpec{
				ClientConfig: v1beta1.ClientConfig{
					Service: &v1beta1.ServiceReference{
						Name:      "my-svc",
						Namespace: "default",
						Path:      "blah",
						Port:      ptr.Int32(8081),
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
		it := &v1beta1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: v1beta1.ClusterInterceptorSpec{
				ClientConfig: v1beta1.ClientConfig{
					URL:     nil,
					Service: nil,
				},
			},
		}
		_, err := it.ResolveAddress()
		if !errors.Is(err, v1beta1.ErrNilURL) {
			t.Fatalf("ResolveToURL expected error to be %s but got %s", v1beta1.ErrNilURL, err)
		}
	})
}
