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

package v1alpha1_test

import (
	"errors"
	"testing"

	"knative.dev/pkg/ptr"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestResolveAddress(t *testing.T) {
	tests := []struct {
		name string
		it   *v1alpha1.ClusterInterceptor
		want string
	}{{
		name: "clientConfig.url is specified",
		it: &v1alpha1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: v1alpha1.ClusterInterceptorSpec{
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
		it: &v1alpha1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: v1alpha1.ClusterInterceptorSpec{
				ClientConfig: v1alpha1.ClientConfig{
					Service: &v1alpha1.ServiceReference{
						Name:      "my-svc",
						Namespace: "default",
						Path:      "blah",
						Port:      ptr.Int32(8888),
					},
				},
			},
		},
		want: "http://my-svc.default.svc:8888/blah",
	}, {
		name: "clientConfig.service without port and scheme so it uses defaultHTTPPort",
		it: &v1alpha1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: v1alpha1.ClusterInterceptorSpec{
				ClientConfig: v1alpha1.ClientConfig{
					Service: &v1alpha1.ServiceReference{
						Name:      "my-svc",
						Namespace: "default",
						Path:      "blah",
					},
				},
			},
		},
		want: "http://my-svc.default.svc:80/blah",
	}, {
		name: "clientConfig with provided caBundle",
		it: &v1alpha1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: v1alpha1.ClusterInterceptorSpec{
				ClientConfig: v1alpha1.ClientConfig{
					CaBundle: []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM5ekNDQXB5Z0F3SUJBZ0lRSllLcEFVeXc2dStvY1JhV1VtRVRoREFLQmdncWhrak9QUVFEQWpCWE1SUXcKRWdZRFZRUUtFd3RyYm1GMGFYWmxMbVJsZGpFL01EMEdBMVVFQXhNMmRHVnJkRzl1TFhSeWFXZG5aWEp6TFdOdgpjbVV0YVc1MFpYSmpaWEIwYjNKekxuUmxhM1J2Ymkxd2FYQmxiR2x1WlhNdWMzWmpNQ0FYRFRJeU1EUXhOVEUyCk1ERTFPRm9ZRHpJeE1qSXdNekl5TVRZd01UVTRXakJYTVJRd0VnWURWUVFLRXd0cmJtRjBhWFpsTG1SbGRqRS8KTUQwR0ExVUVBeE0yZEdWcmRHOXVMWFJ5YVdkblpYSnpMV052Y21VdGFXNTBaWEpqWlhCMGIzSnpMblJsYTNSdgpiaTF3YVhCbGJHbHVaWE11YzNaak1Ga3dFd1lIS29aSXpqMENBUVlJS29aSXpqMERBUWNEUWdBRUFHcHp1RjlQCjY5VnFhN0xIY0tmNGpWY2JqblJNWDAxYWRnakh0Zy9kZFdIaVBWdXVJZER1WnZzVTREaVp5Smh2WnpmaHQ0ZmsKT3FJc3dJeVlmbkpLRnFPQ0FVWXdnZ0ZDTUE0R0ExVWREd0VCL3dRRUF3SUNoREFkQmdOVkhTVUVGakFVQmdncgpCZ0VGQlFjREFRWUlLd1lCQlFVSEF3SXdEd1lEVlIwVEFRSC9CQVV3QXdFQi96QWRCZ05WSFE0RUZnUVVQRXFjCnEvRFJHd2FDUTdmOFc0dmlucGN5a09zd2dlQUdBMVVkRVFTQjJEQ0IxWUloZEdWcmRHOXVMWFJ5YVdkblpYSnoKTFdOdmNtVXRhVzUwWlhKalpYQjBiM0p6Z2pKMFpXdDBiMjR0ZEhKcFoyZGxjbk10WTI5eVpTMXBiblJsY21ObApjSFJ2Y25NdWRHVnJkRzl1TFhCcGNHVnNhVzVsYzRJMmRHVnJkRzl1TFhSeWFXZG5aWEp6TFdOdmNtVXRhVzUwClpYSmpaWEIwYjNKekxuUmxhM1J2Ymkxd2FYQmxiR2x1WlhNdWMzWmpna1IwWld0MGIyNHRkSEpwWjJkbGNuTXQKWTI5eVpTMXBiblJsY21ObGNIUnZjbk11ZEdWcmRHOXVMWEJwY0dWc2FXNWxjeTV6ZG1NdVkyeDFjM1JsY2k1cwpiMk5oYkRBS0JnZ3Foa2pPUFFRREFnTkpBREJHQWlFQTlhWFBtUFZzRVA3R0xTbzI0SnNmNnRGTmpyQWJRbEl0CjRCYXllcjBnaU5jQ0lRQ09XSm1NTXQxQkE1RXgwa0FYTWRtZjlFdXV4LzlyUUkzMm9VNjVSYm9mNEE9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="),
					Service: &v1alpha1.ServiceReference{
						Name:      "my-svc",
						Namespace: "default",
						Path:      "blah",
					},
				},
			},
		},
		want: "https://my-svc.default.svc:8443/blah",
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
		it := &v1alpha1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: v1alpha1.ClusterInterceptorSpec{
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
