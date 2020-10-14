/*
Copyright 2020 The Tekton Authors

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
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"knative.dev/pkg/ptr"
)

func TestTriggerSetDefaults(t *testing.T) {
	tests := []struct {
		name string
		in   *v1alpha1.Trigger
		want *v1alpha1.Trigger
		wc   func(context.Context) context.Context
	}{{
		name: "default binding kind",
		in: &v1alpha1.Trigger{
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Ref: "binding",
				}, {
					Kind: v1alpha1.NamespacedTriggerBindingKind,
					Ref:  "namespace-binding",
				}, {
					Kind: v1alpha1.ClusterTriggerBindingKind,
					Ref:  "cluster-binding",
				}},
			},
		},
		wc: v1alpha1.WithUpgradeViaDefaulting,
		want: &v1alpha1.Trigger{
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Kind: v1alpha1.NamespacedTriggerBindingKind,
					Ref:  "binding",
				}, {
					Kind: v1alpha1.NamespacedTriggerBindingKind,
					Ref:  "namespace-binding",
				}, {
					Kind: v1alpha1.ClusterTriggerBindingKind,
					Ref:  "cluster-binding",
				}},
			},
		},
	}, {
		name: "default old embed binding to new",
		in: &v1alpha1.Trigger{
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Kind: v1alpha1.NamespacedTriggerBindingKind,
					Ref:  "binding",
				}, {
					Kind: v1alpha1.NamespacedTriggerBindingKind,
					Spec: &v1alpha1.TriggerBindingSpec{
						Params: []v1alpha1.Param{{
							Name:  "p1",
							Value: "v1",
						}, {
							Name:  "p2",
							Value: "v2",
						}},
					},
				}},
			},
		},
		wc: v1alpha1.WithUpgradeViaDefaulting,
		want: &v1alpha1.Trigger{
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Kind: v1alpha1.NamespacedTriggerBindingKind,
					Ref:  "binding",
				}, {
					Name:  "p1",
					Value: ptr.String("v1"),
				}, {
					Name:  "p2",
					Value: ptr.String("v2"),
				}},
			},
		},
	}, {
		name: "upgrade context not set",
		in: &v1alpha1.Trigger{
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Ref: "binding",
				}},
			},
		},
		want: &v1alpha1.Trigger{
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Ref: "binding", // If upgrade context was set, Kind should have been added
				}},
			},
		},
	}, {
		name: "sets template name to ref",
		wc:   v1alpha1.WithUpgradeViaDefaulting,
		in: &v1alpha1.Trigger{
			Spec: v1alpha1.TriggerSpec{
				Template: v1alpha1.TriggerSpecTemplate{
					Name: "tt-name",
				},
			},
		},
		want: &v1alpha1.Trigger{
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{},
				Template: v1alpha1.TriggerSpecTemplate{
					Ref:  ptr.String("tt-name"),
					Name: "",
				},
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in
			ctx := context.Background()
			if tc.wc != nil {
				ctx = tc.wc(ctx)
			}
			got.SetDefaults(ctx)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("SetDefaults (-want, +got) = %v", diff)
			}
		})
	}
}
