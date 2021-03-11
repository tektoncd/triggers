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
	"context"
	"testing"

	"knative.dev/pkg/ptr"

	"github.com/google/go-cmp/cmp"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterInterceptorSetDefaults(t *testing.T) {
	tests := []struct {
		name string
		in   triggersv1.ClusterInterceptor
		want triggersv1.ClusterInterceptor
	}{{
		name: "sets default service port",
		in: triggersv1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "github",
			},
			Spec: triggersv1.ClusterInterceptorSpec{
				ClientConfig: triggersv1.ClientConfig{
					Service: &triggersv1.ServiceReference{
						Namespace: "default",
						Name:      "github-svc",
					},
				},
			},
		},
		want: triggersv1.ClusterInterceptor{
			ObjectMeta: metav1.ObjectMeta{
				Name: "github",
			},
			Spec: triggersv1.ClusterInterceptorSpec{
				ClientConfig: triggersv1.ClientConfig{
					Service: &triggersv1.ServiceReference{
						Namespace: "default",
						Name:      "github-svc",
						Port:      ptr.Int32(80),
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
				t.Fatalf("ClusterInterceptor SetDefaults error: %s", diff)
			}
		})
	}
}
