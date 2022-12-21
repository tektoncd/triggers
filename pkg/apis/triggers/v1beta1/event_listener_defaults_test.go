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
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/config"
	"github.com/tektoncd/triggers/pkg/apis/triggers/contexts"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/ptr"
)

func TestEventListenerSetDefaults(t *testing.T) {
	tests := []struct {
		name string
		in   *v1beta1.EventListener
		want *v1beta1.EventListener
		wc   func(context.Context) context.Context
	}{{
		name: "default binding",
		in: &v1beta1.EventListener{
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{
						{
							Ref: "binding",
						},
						{
							Kind: v1beta1.NamespacedTriggerBindingKind,
							Ref:  "namespace-binding",
						},
						{
							Kind: v1beta1.ClusterTriggerBindingKind,
							Ref:  "cluster-binding",
						},
					},
				}},
			},
		},
		wc: contexts.WithUpgradeViaDefaulting,
		want: &v1beta1.EventListener{
			Spec: v1beta1.EventListenerSpec{
				ServiceAccountName: config.DefaultServiceAccountValue,
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{
						{
							Kind: v1beta1.NamespacedTriggerBindingKind,
							Ref:  "binding",
						},
						{
							Kind: v1beta1.NamespacedTriggerBindingKind,
							Ref:  "namespace-binding",
						},
						{
							Kind: v1beta1.ClusterTriggerBindingKind,
							Ref:  "cluster-binding",
						},
					},
				}},
			},
		},
	}, {
		name: "set replicas to 1 if provided replicas is 0 as part of eventlistener kubernetesResources",
		in: &v1beta1.EventListener{
			Spec: v1beta1.EventListenerSpec{
				Resources: v1beta1.Resources{
					KubernetesResource: &v1beta1.KubernetesResource{
						Replicas: ptr.Int32(0),
					},
				},
			},
		},
		wc: contexts.WithUpgradeViaDefaulting,
		want: &v1beta1.EventListener{
			Spec: v1beta1.EventListenerSpec{
				ServiceAccountName: config.DefaultServiceAccountValue,
				Resources: v1beta1.Resources{
					KubernetesResource: &v1beta1.KubernetesResource{
						Replicas: ptr.Int32(1),
					},
				},
			},
		},
	}, {
		name: "different value for replicas other than 0",
		in: &v1beta1.EventListener{
			Spec: v1beta1.EventListenerSpec{
				Resources: v1beta1.Resources{
					KubernetesResource: &v1beta1.KubernetesResource{
						Replicas: ptr.Int32(2),
					},
				},
			},
		},
		wc: contexts.WithUpgradeViaDefaulting,
		want: &v1beta1.EventListener{
			Spec: v1beta1.EventListenerSpec{
				ServiceAccountName: config.DefaultServiceAccountValue,
				Resources: v1beta1.Resources{
					KubernetesResource: &v1beta1.KubernetesResource{
						Replicas: ptr.Int32(2),
					},
				},
			},
		},
	}, {
		name: "adds interceptorkind when not specified",
		in: &v1beta1.EventListener{
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Interceptors: []*v1beta1.EventInterceptor{{
						Ref: v1beta1.InterceptorRef{
							Name: "cel",
						},
					}},
				}},
			},
		},
		wc: contexts.WithUpgradeViaDefaulting,
		want: &v1beta1.EventListener{
			Spec: v1beta1.EventListenerSpec{
				ServiceAccountName: config.DefaultServiceAccountValue,
				Triggers: []v1beta1.EventListenerTrigger{{
					Interceptors: []*v1beta1.EventInterceptor{{
						Ref: v1beta1.InterceptorRef{
							Name: "cel",
							Kind: v1beta1.ClusterInterceptorKind,
						},
					}},
				}},
			},
		},
	}, {
		name: "EventListener default config context with sa",
		in: &v1beta1.EventListener{
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Interceptors: []*v1beta1.EventInterceptor{{
						Ref: v1beta1.InterceptorRef{
							Name: "cel",
						},
					}},
				}},
			},
		},
		want: &v1beta1.EventListener{
			Spec: v1beta1.EventListenerSpec{
				ServiceAccountName: "tekton",
				Triggers: []v1beta1.EventListenerTrigger{{
					Interceptors: []*v1beta1.EventInterceptor{{
						Ref: v1beta1.InterceptorRef{
							Name: "cel",
							Kind: v1beta1.ClusterInterceptorKind,
						},
					}},
				}},
			},
		},
		wc: func(ctx context.Context) context.Context {
			s := config.NewStore(logtesting.TestLogger(t))
			s.OnConfigChanged(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: config.GetDefaultsConfigName(),
				},
				Data: map[string]string{
					"default-service-account": "tekton",
				},
			})
			return contexts.WithUpgradeViaDefaulting(s.ToContext(ctx))
		},
	}, {
		name: "adds TriggerGroup interceptorkind when not specified",
		in: &v1beta1.EventListener{
			Spec: v1beta1.EventListenerSpec{
				TriggerGroups: []v1beta1.EventListenerTriggerGroup{{
					Interceptors: []*v1beta1.EventInterceptor{{
						Ref: v1beta1.InterceptorRef{
							Name: "cel",
						},
					}},
				}},
			},
		},
		wc: contexts.WithUpgradeViaDefaulting,
		want: &v1beta1.EventListener{
			Spec: v1beta1.EventListenerSpec{
				ServiceAccountName: config.DefaultServiceAccountValue,
				TriggerGroups: []v1beta1.EventListenerTriggerGroup{{
					Interceptors: []*v1beta1.EventInterceptor{{
						Ref: v1beta1.InterceptorRef{
							Name: "cel",
							Kind: v1beta1.ClusterInterceptorKind,
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
