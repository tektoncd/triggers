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

package clusterinterceptor

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers/contexts"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/ptr"
)

func TestReconcileKind(t *testing.T) {
	tests := []struct {
		name    string
		initial *triggersv1.ClusterInterceptor // State of the world before we call Reconcile
		want    *triggersv1.ClusterInterceptor // Expected State of the world after calling Reconcile
	}{{
		name: "inital status is nil",
		initial: &triggersv1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: triggersv1.ClusterInterceptorSpec{
				ClientConfig: triggersv1.ClientConfig{
					Service: &triggersv1.ServiceReference{
						Name:      "my-svc",
						Namespace: "default",
						Path:      "path",
						Port:      ptr.Int32(80),
					},
				}},
			Status: triggersv1.ClusterInterceptorStatus{},
		},
		want: &triggersv1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: triggersv1.ClusterInterceptorSpec{
				ClientConfig: triggersv1.ClientConfig{
					Service: &triggersv1.ServiceReference{
						Name:      "my-svc",
						Namespace: "default",
						Path:      "path",
						Port:      ptr.Int32(80),
					},
				}},
			Status: triggersv1.ClusterInterceptorStatus{
				AddressStatus: duckv1.AddressStatus{
					Address: &duckv1.Addressable{
						URL: &apis.URL{
							Scheme: "http",
							Host:   "my-svc.default.svc:80",
							Path:   "path",
						},
					},
				},
			},
		},
	}, {
		name: "defaults are applied",
		initial: &triggersv1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: triggersv1.ClusterInterceptorSpec{
				ClientConfig: triggersv1.ClientConfig{
					Service: &triggersv1.ServiceReference{
						Name:      "my-svc",
						Namespace: "default",
						Path:      "path",
					},
				}},
			Status: triggersv1.ClusterInterceptorStatus{},
		},
		want: &triggersv1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-interceptor",
			},
			Spec: triggersv1.ClusterInterceptorSpec{
				ClientConfig: triggersv1.ClientConfig{
					Service: &triggersv1.ServiceReference{
						Name:      "my-svc",
						Namespace: "default",
						Path:      "path",
						Port:      ptr.Int32(80),
					},
				}},
			Status: triggersv1.ClusterInterceptorStatus{
				AddressStatus: duckv1.AddressStatus{
					Address: &duckv1.Addressable{
						URL: &apis.URL{
							Scheme: "http",
							Host:   "my-svc.default.svc:80",
							Path:   "path",
						},
					},
				},
			},
		},
	}}

	for _, tc := range tests {
		r := Reconciler{}
		context := contexts.WithUpgradeViaDefaulting(logtesting.TestContextWithLogger(t))
		err := r.ReconcileKind(context, tc.initial)
		if err != nil {
			t.Fatalf("ReconcileKind() unexpected error: %v", err)
		}
		got := tc.initial
		if diff := cmp.Diff(tc.want, got); diff != "" {
			t.Fatalf("ReconcileKind() diff -want/+got: %s", diff)
		}
	}
}
