/*
Copyright 2019 The Tekton Authors

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
	"github.com/tektoncd/triggers/pkg/apis/triggers/contexts"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/ptr"
)

func TestEventListenerSetDefaults(t *testing.T) {
	tests := []struct {
		name string
		in   *v1alpha1.EventListener
		want *v1alpha1.EventListener
		wc   func(context.Context) context.Context
	}{{
		name: "default binding",
		in: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{
						{
							Ref: "binding",
						},
						{
							Kind: v1alpha1.NamespacedTriggerBindingKind,
							Ref:  "namespace-binding",
						},
						{
							Kind: v1alpha1.ClusterTriggerBindingKind,
							Ref:  "cluster-binding",
						},
					},
				}},
			},
		},
		wc: contexts.WithUpgradeViaDefaulting,
		want: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{
						{
							Kind: v1alpha1.NamespacedTriggerBindingKind,
							Ref:  "binding",
						},
						{
							Kind: v1alpha1.NamespacedTriggerBindingKind,
							Ref:  "namespace-binding",
						},
						{
							Kind: v1alpha1.ClusterTriggerBindingKind,
							Ref:  "cluster-binding",
						},
					},
				}},
			},
		},
	}, {
		name: "set replicas to 1 if provided deprecated replicas is 0 as part of eventlistener spec",
		in: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				DeprecatedReplicas: ptr.Int32(0),
			},
		},
		wc: contexts.WithUpgradeViaDefaulting,
		want: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
						Replicas: ptr.Int32(1),
					},
				},
			},
		},
	}, {
		name: "set replicas to 1 if provided replicas is 0 as part of eventlistener kubernetesResources",
		in: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
						Replicas: ptr.Int32(0),
					},
				},
			},
		},
		wc: contexts.WithUpgradeViaDefaulting,
		want: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
						Replicas: ptr.Int32(1),
					},
				},
			},
		},
	}, {
		name: "deprecate podTemplate nodeselector to resource",
		in: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				DeprecatedPodTemplate: &v1alpha1.PodTemplate{
					NodeSelector: map[string]string{"beta.kubernetes.io/os": "linux"},
				},
			},
		},
		wc: contexts.WithUpgradeViaDefaulting,
		want: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
						WithPodSpec: duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								Spec: corev1.PodSpec{
									NodeSelector: map[string]string{"beta.kubernetes.io/os": "linux"},
								},
							},
						}},
				},
			},
		},
	}, {
		name: "deprecate podTemplate toleration to resource",
		in: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				DeprecatedPodTemplate: &v1alpha1.PodTemplate{Tolerations: []corev1.Toleration{{Key: "key"}}},
			},
		},
		wc: contexts.WithUpgradeViaDefaulting,
		want: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
						WithPodSpec: duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								Spec: corev1.PodSpec{Tolerations: []corev1.Toleration{{Key: "key"}}},
							},
						}},
				},
			},
		},
	}, {
		name: "different value for replicas other than 0",
		in: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
						Replicas: ptr.Int32(2),
					},
				},
			},
		},
		wc: contexts.WithUpgradeViaDefaulting,
		want: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
						Replicas: ptr.Int32(2),
					},
				},
			},
		},
	}, {
		name: "different value for deprecated replicas other than 0",
		in: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				DeprecatedReplicas: ptr.Int32(2),
			},
		},
		wc: contexts.WithUpgradeViaDefaulting,
		want: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
						Replicas: ptr.Int32(2),
					},
				},
			},
		},
	}, {
		name: "adds interceptorkind when not specified",
		in: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Interceptors: []*v1alpha1.EventInterceptor{{
						Ref: v1alpha1.InterceptorRef{
							Name: "cel",
						},
					}},
				}},
			},
		},
		wc: contexts.WithUpgradeViaDefaulting,
		want: &v1alpha1.EventListener{
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Interceptors: []*v1alpha1.EventInterceptor{{
						Ref: v1alpha1.InterceptorRef{
							Name: "cel",
							Kind: v1alpha1.ClusterInterceptorKind,
						},
					}},
				}},
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
