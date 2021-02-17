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

package builder

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/ptr"
)

func TestEventListenerBuilder(t *testing.T) {
	tests := []struct {
		name    string
		normal  *v1alpha1.EventListener
		builder *v1alpha1.EventListener
	}{{
		name:    "Empty",
		normal:  &v1alpha1.EventListener{},
		builder: EventListener("", ""),
	}, {
		name: "Name and Namespace",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
		},
		builder: EventListener("name", "namespace"),
	}, {
		name: "No Triggers",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				ServiceAccountName: "serviceAccount",
			},
		},
		builder: EventListener("name", "namespace",
			EventListenerSpec(
				EventListenerServiceAccount("serviceAccount"),
			),
		),
	}, {
		name: "Status configuration",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Status: v1alpha1.EventListenerStatus{
				AddressStatus: duckv1alpha1.AddressStatus{
					Address: NewAddressable("hostname"),
				},
				Configuration: v1alpha1.EventListenerConfig{
					GeneratedResourceName: "generatedName",
				},
			},
		},
		builder: EventListener("name", "namespace",
			EventListenerStatus(
				EventListenerConfig("generatedName"),
				EventListenerAddress("hostname"),
			),
		),
	}, {
		name: "One Condition",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Status: v1alpha1.EventListenerStatus{
				Status: duckv1.Status{
					Conditions: []apis.Condition{{
						Type:    v1alpha1.ServiceExists,
						Status:  corev1.ConditionTrue,
						Message: "Service exists",
					}},
				},
			},
		},
		builder: EventListener("name", "namespace",
			EventListenerStatus(
				EventListenerCondition(
					v1alpha1.ServiceExists,
					corev1.ConditionTrue,
					"Service exists", "",
				),
			),
		),
	}, {
		name: "Two Condition",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Status: v1alpha1.EventListenerStatus{
				Status: duckv1.Status{
					Conditions: []apis.Condition{{
						Type:    v1alpha1.DeploymentExists,
						Status:  corev1.ConditionTrue,
						Message: "Deployment exists",
					}, {
						Type:    v1alpha1.ServiceExists,
						Status:  corev1.ConditionTrue,
						Message: "Service exists",
					}},
				},
			},
		},
		builder: EventListener("name", "namespace",
			EventListenerStatus(
				EventListenerCondition(
					v1alpha1.ServiceExists,
					corev1.ConditionTrue,
					"Service exists", "",
				),
				EventListenerCondition(
					v1alpha1.DeploymentExists,
					corev1.ConditionTrue,
					"Deployment exists", "",
				),
			),
		),
	}, {
		name: "One Trigger with one TriggerRef",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				ServiceAccountName: "serviceAccount",
				Triggers: []v1alpha1.EventListenerTrigger{{
					TriggerRef: "my-trigger",
				}},
			},
		},
		builder: EventListener("name", "namespace",
			EventListenerSpec(
				EventListenerServiceAccount("serviceAccount"),
				EventListenerTriggerRef("my-trigger"))),
	}, {
		name: "EventListener with Matchnames NamespaceSelector",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				ServiceAccountName: "serviceAccount",
				Triggers: []v1alpha1.EventListenerTrigger{{
					TriggerRef: "my-trigger",
				}},
				NamespaceSelector: v1alpha1.NamespaceSelector{
					MatchNames: []string{"foo", "bar"},
				},
			},
		},
		builder: EventListener("name", "namespace",
			EventListenerSpec(
				EventListenerNamespaceSelectorMatchNames([]string{"foo", "bar"}),
				EventListenerServiceAccount("serviceAccount"),
				EventListenerTriggerRef("my-trigger"))),
	}, {
		name: "One Trigger with one Binding",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				ServiceAccountName: "serviceAccount",
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						Ref:        "tb1",
						APIVersion: "v1alpha1",
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref:        ptr.String("tt1"),
						APIVersion: "v1alpha1",
					},
				}},
			},
		},
		builder: EventListener("name", "namespace",
			EventListenerSpec(
				EventListenerServiceAccount("serviceAccount"),
				EventListenerTrigger("tt1", "v1alpha1",
					EventListenerTriggerBinding("tb1", "", "v1alpha1"),
				),
			),
		),
	}, {
		name: "One Trigger with one TriggerBinding",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				ServiceAccountName: "serviceAccount",
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						Ref:        "tb1",
						APIVersion: "v1alpha1",
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref:        ptr.String("tt1"),
						APIVersion: "v1alpha1",
					},
				}},
			},
		},
		builder: EventListener("name", "namespace",
			EventListenerSpec(
				EventListenerServiceAccount("serviceAccount"),
				EventListenerTrigger("tt1", "v1alpha1",
					EventListenerTriggerBinding("tb1", "TriggerBinding", "v1alpha1"),
				),
			),
		),
	}, {
		name: "One Trigger with ClusterTriggerBinding",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				ServiceAccountName: "serviceAccount",
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{
						Kind:       v1alpha1.ClusterTriggerBindingKind,
						Ref:        "tb1",
						APIVersion: "v1alpha1",
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref:        ptr.String("tt1"),
						APIVersion: "v1alpha1",
					},
				}},
			},
		},
		builder: EventListener("name", "namespace",
			EventListenerSpec(
				EventListenerServiceAccount("serviceAccount"),
				EventListenerTrigger("tt1", "v1alpha1",
					EventListenerTriggerBinding("tb1", "ClusterTriggerBinding", "v1alpha1"),
				),
			),
		),
	}, {
		name: "One Trigger with multiple TriggerBindings",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				ServiceAccountName: "serviceAccount",
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{
						{
							Kind:       v1alpha1.NamespacedTriggerBindingKind,
							Ref:        "tb1",
							APIVersion: "v1alpha1",
						},
						{
							Kind:       v1alpha1.ClusterTriggerBindingKind,
							Ref:        "ctb1",
							APIVersion: "v1alpha1",
						},
					},
					Template: &v1alpha1.EventListenerTemplate{
						Ref:        ptr.String("tt1"),
						APIVersion: "v1alpha1",
					},
				}},
			},
		},
		builder: EventListener("name", "namespace",
			EventListenerSpec(
				EventListenerServiceAccount("serviceAccount"),
				EventListenerTrigger("tt1", "v1alpha1",
					EventListenerTriggerBinding("tb1", "", "v1alpha1"),
					EventListenerTriggerBinding("ctb1", "ClusterTriggerBinding", "v1alpha1"),
				),
			),
		),
	}, {
		name: "Two Trigger with extra Meta",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
				Labels: map[string]string{
					"key": "value",
				},
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "EventListener",
				APIVersion: "v1alpha1",
			},
			Spec: v1alpha1.EventListenerSpec{
				ServiceAccountName: "serviceAccount",
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						Ref:        "tb1",
						APIVersion: "v1alpha1",
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref:        ptr.String("tt1"),
						APIVersion: "v1alpha1",
					},
				}, {
					Bindings: []*v1alpha1.EventListenerBinding{{
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						Ref:        "tb2",
						APIVersion: "v1alpha1",
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref:        ptr.String("tt2"),
						APIVersion: "v1alpha1",
					},
				},
				},
			},
		},
		builder: EventListener("name", "namespace",
			EventListenerMeta(
				TypeMeta("EventListener", "v1alpha1"),
				Label("key", "value"),
			),
			EventListenerSpec(
				EventListenerServiceAccount("serviceAccount"),
				EventListenerTrigger("tt1", "v1alpha1",
					EventListenerTriggerBinding("tb1", "", "v1alpha1"),
				),
				EventListenerTrigger("tt2", "v1alpha1",
					EventListenerTriggerBinding("tb2", "", "v1alpha1"),
				),
			),
		),
	}, {
		name: "One Trigger with Interceptor",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				ServiceAccountName: "serviceAccount",
				Triggers: []v1alpha1.EventListenerTrigger{{
					Name: "foo-trig",
					Interceptors: []*v1alpha1.EventInterceptor{{
						Webhook: &v1alpha1.WebhookInterceptor{
							ObjectRef: &corev1.ObjectReference{
								Kind:       "Service",
								Namespace:  "namespace",
								Name:       "foo",
								APIVersion: "v1",
							},
						},
					}},
					Bindings: []*v1alpha1.EventListenerBinding{{
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						Ref:        "tb1",
						APIVersion: "v1alpha1",
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref:        ptr.String("tt1"),
						APIVersion: "v1alpha1",
					},
				}},
			},
		},
		builder: EventListener("name", "namespace",
			EventListenerSpec(
				EventListenerServiceAccount("serviceAccount"),
				EventListenerTrigger("tt1", "v1alpha1",
					EventListenerTriggerBinding("tb1", "", "v1alpha1"),
					EventListenerTriggerName("foo-trig"),
					EventListenerTriggerInterceptor("foo", "v1", "Service", "namespace"),
				),
			),
		),
	}, {
		name: "One Trigger with Interceptor With Header",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				ServiceAccountName: "serviceAccount",
				Triggers: []v1alpha1.EventListenerTrigger{{
					Name: "foo-trig",
					Interceptors: []*v1alpha1.EventInterceptor{{
						Webhook: &v1alpha1.WebhookInterceptor{
							ObjectRef: &corev1.ObjectReference{
								Kind:       "Service",
								Namespace:  "namespace",
								Name:       "foo",
								APIVersion: "v1",
							},
							Header: []pipelinev1.Param{
								{
									Name: "header1",
									Value: pipelinev1.ArrayOrString{
										ArrayVal: []string{"value1"},
										Type:     pipelinev1.ParamTypeArray,
									},
								},
							},
						},
					}},
					Bindings: []*v1alpha1.EventListenerBinding{{
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						Ref:        "tb1",
						APIVersion: "v1alpha1",
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref:        ptr.String("tt1"),
						APIVersion: "v1alpha1",
					},
				}},
			}},
		builder: EventListener("name", "namespace",
			EventListenerSpec(
				EventListenerServiceAccount("serviceAccount"),
				EventListenerTrigger("tt1", "v1alpha1",
					EventListenerTriggerBinding("tb1", "", "v1alpha1"),
					EventListenerTriggerName("foo-trig"),
					EventListenerTriggerInterceptor("foo", "v1", "Service", "namespace",
						EventInterceptorParam("header1", "value1"),
					),
				),
			),
		),
	}, {
		name: "One Trigger with DeprecatedCEL Interceptor",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				ServiceAccountName: "serviceAccount",
				Triggers: []v1alpha1.EventListenerTrigger{{
					Name: "foo-trig",
					Interceptors: []*v1alpha1.EventInterceptor{{
						DeprecatedCEL: &v1alpha1.CELInterceptor{
							Filter: "body.value == 'test'",
							Overlays: []v1alpha1.CELOverlay{
								{Key: "value", Expression: "'testing'"},
							},
						},
					}},
					Bindings: []*v1alpha1.EventListenerBinding{{
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						Ref:        "tb1",
						APIVersion: "v1alpha1",
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref:        ptr.String("tt1"),
						APIVersion: "v1alpha1",
					},
				}},
			},
		},
		builder: EventListener("name", "namespace",
			EventListenerSpec(
				EventListenerServiceAccount("serviceAccount"),
				EventListenerTrigger("tt1", "v1alpha1",
					EventListenerTriggerBinding("tb1", "", "v1alpha1"),
					EventListenerTriggerName("foo-trig"),
					EventListenerCELInterceptor("body.value == 'test'", EventListenerCELOverlay("value", "'testing'")),
				),
			),
		),
	}, {
		name: "Eventlistener with kubernetes resource",
		normal: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Name: "foo-trig",
					Bindings: []*v1alpha1.EventListenerBinding{{
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						Ref:        "tb1",
						APIVersion: "v1alpha1",
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref:        ptr.String("tt1"),
						APIVersion: "v1alpha1",
					},
				}},
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
						ServiceType: "NodePort",
						WithPodSpec: duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								ObjectMeta: metav1.ObjectMeta{
									Labels:      map[string]string{"key": "value"},
									Annotations: map[string]string{"key": "value"},
								},
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Env: []corev1.EnvVar{{
											Name: "HTTP",
										}},
									}},
									NodeSelector:       map[string]string{"beta.kubernetes.io/os": "linux"},
									ServiceAccountName: "resource",
									Tolerations: []corev1.Toleration{{
										Key:      "key",
										Operator: "Equal",
										Value:    "value",
										Effect:   "NoSchedule",
									}},
								},
							},
						},
					}},
			},
		},
		builder: EventListener("name", "namespace",
			EventListenerSpec(
				EventListenerResources(
					EventListenerKubernetesResources(
						EventListenerServiceType("NodePort"),
						EventListenerPodSpec(duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								ObjectMeta: metav1.ObjectMeta{
									Labels:      map[string]string{"key": "value"},
									Annotations: map[string]string{"key": "value"},
								},
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Env: []corev1.EnvVar{{
											Name: "HTTP",
										}},
									}},
									NodeSelector:       map[string]string{"beta.kubernetes.io/os": "linux"},
									ServiceAccountName: "resource",
									Tolerations: []corev1.Toleration{{
										Key:      "key",
										Operator: "Equal",
										Value:    "value",
										Effect:   "NoSchedule",
									}},
								},
							},
						}),
					),
				),
				EventListenerTrigger("tt1", "v1alpha1",
					EventListenerTriggerBinding("tb1", "", "v1alpha1"),
					EventListenerTriggerName("foo-trig"),
				),
			),
		),
	},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.normal, tt.builder, cmpopts.IgnoreTypes(apis.Condition{}.LastTransitionTime.Inner.Time)); diff != "" {
				t.Errorf("EventListener() builder equality mismatch. Diff request body: -want +got: %s", diff)
			}
		})
	}
}
