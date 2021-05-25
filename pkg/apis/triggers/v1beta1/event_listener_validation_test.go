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

package v1beta1_test

import (
	"context"
	"testing"

	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/ptr"
)

func Test_EventListenerValidate_OnDelete(t *testing.T) {
	el := &triggersv1beta1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: triggersv1beta1.EventListenerSpec{
			Triggers: []triggersv1beta1.EventListenerTrigger{{
				Template: &triggersv1beta1.EventListenerTemplate{
					Ref: ptr.String(""),
				},
			}},
		},
	}
	err := el.Validate(apis.WithinDelete(context.Background()))
	if err != nil {
		t.Errorf("EventListener.Validate() on Delete expected no error, but got one, EventListener: %v, error: %v", el, err)
	}
}

func Test_EventListenerValidate(t *testing.T) {
	tests := []struct {
		name string
		el   *triggersv1beta1.EventListener
	}{{
		name: "TriggerTemplate Does Not Exist",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref:        ptr.String("dne"),
						APIVersion: "v1beta1",
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener No TriggerBinding",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref:        ptr.String("tt"),
						APIVersion: "v1beta1",
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with TriggerRef",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					TriggerRef: "tt",
				}},
			},
		},
	}, {
		name: "Valid EventListener with TriggerBinding",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       triggersv1beta1.NamespacedTriggerBindingKind,
						APIVersion: "triggersv1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with ClusterTriggerBinding",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       triggersv1beta1.ClusterTriggerBindingKind,
						APIVersion: "v1alpha1",
					}},
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with multiple TriggerBindings",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{
						Ref:        "tb1",
						Kind:       triggersv1beta1.ClusterTriggerBindingKind,
						APIVersion: "triggersv1beta1", // TODO: APIVersions seem wrong?
					}, {
						Ref:        "tb2",
						Kind:       triggersv1beta1.NamespacedTriggerBindingKind,
						APIVersion: "triggersv1beta1", // TODO: APIVersions seem wrong?
					}, {
						Ref:        "tb3",
						Kind:       triggersv1beta1.NamespacedTriggerBindingKind,
						APIVersion: "triggersv1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener Webhook Interceptor",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Webhook: &triggersv1beta1.WebhookInterceptor{
							ObjectRef: &corev1.ObjectReference{
								Name:       "svc",
								Kind:       "Service",
								APIVersion: "v1",
								Namespace:  "namespace",
							},
						},
					}},
					Bindings: []*triggersv1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       triggersv1beta1.NamespacedTriggerBindingKind,
						APIVersion: "triggersv1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener Interceptor With Header",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Webhook: &triggersv1beta1.WebhookInterceptor{
							Header: []pipelinev1alpha1.Param{{
								Name: "Valid-Header-Key",
								Value: pipelinev1alpha1.ArrayOrString{
									Type:      pipelinev1alpha1.ParamTypeString,
									StringVal: "valid-value",
								},
							}, {
								Name: "Valid-Header-Key2",
								Value: pipelinev1alpha1.ArrayOrString{
									Type:      pipelinev1alpha1.ParamTypeString,
									StringVal: "valid value 2",
								},
							}},
							ObjectRef: &corev1.ObjectReference{
								Name:       "svc",
								Kind:       "Service",
								APIVersion: "v1",
								Namespace:  "namespace",
							},
						},
					}},
					Bindings: []*triggersv1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       triggersv1beta1.NamespacedTriggerBindingKind,
						APIVersion: "triggersv1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener Two Triggers",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       triggersv1beta1.NamespacedTriggerBindingKind,
						APIVersion: "triggersv1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}, {
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
					Bindings: []*triggersv1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       triggersv1beta1.NamespacedTriggerBindingKind,
						APIVersion: "triggersv1beta1", // TODO: APIVersions seem wrong?
					}},
				}},
			},
		},
	}, {
		name: "Valid EventListener with CEL interceptor",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Ref: triggersv1beta1.InterceptorRef{
							Name:       "cel",
							Kind:       triggersv1beta1.ClusterInterceptorKind,
							APIVersion: "triggers.tekton.dev/triggersv1beta1",
						},
						Params: []triggersv1beta1.InterceptorParams{{
							Name:  "filter",
							Value: test.ToV1JSON(t, "body.value == test"),
						}, {
							Name: "overlays",
							Value: test.ToV1JSON(t, []triggersv1beta1.CELOverlay{{
								Key:        "value",
								Expression: "testing",
							}}),
						}},
					}},
					Bindings: []*triggersv1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       triggersv1beta1.NamespacedTriggerBindingKind,
						APIVersion: "triggersv1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with no trigger name",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       triggersv1beta1.NamespacedTriggerBindingKind,
						APIVersion: "triggersv1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "Namespace selector with label selector",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				NamespaceSelector: triggersv1beta1.NamespaceSelector{
					MatchNames: []string{"foo"},
				},
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"foo": "bar",
					},
				},
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Ref: "bindingRef", Kind: triggersv1beta1.NamespacedTriggerBindingKind}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
				}},
			},
		},
	}, {
		name: "Valid EventListener with kubernetes env for podspec",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
				Resources: triggersv1beta1.Resources{
					KubernetesResource: &triggersv1beta1.KubernetesResource{
						WithPodSpec: duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								Spec: corev1.PodSpec{
									ServiceAccountName: "k8sresource",
									Tolerations: []corev1.Toleration{{
										Key:      "key",
										Operator: "Equal",
										Value:    "value",
										Effect:   "NoSchedule",
									}},
									NodeSelector: map[string]string{"beta.kubernetes.io/os": "linux"},
									Containers: []corev1.Container{{
										Resources: corev1.ResourceRequirements{
											Limits: corev1.ResourceList{
												corev1.ResourceCPU:    resource.Quantity{Format: resource.DecimalSI},
												corev1.ResourceMemory: resource.Quantity{Format: resource.BinarySI},
											},
											Requests: corev1.ResourceList{
												corev1.ResourceCPU:    resource.Quantity{Format: resource.DecimalSI},
												corev1.ResourceMemory: resource.Quantity{Format: resource.BinarySI},
											},
										},
									}},
								},
							},
						},
					},
				},
			},
		},
	}, {
		name: "Valid Replicas for EventListener",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Resources: triggersv1beta1.Resources{
					KubernetesResource: &triggersv1beta1.KubernetesResource{
						Replicas: ptr.Int32(1),
					},
				},
			},
		},
	}, {
		name: "Valid EventListener with env for TLS connection",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
				Resources: triggersv1beta1.Resources{
					KubernetesResource: &triggersv1beta1.KubernetesResource{
						WithPodSpec: duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								Spec: corev1.PodSpec{
									ServiceAccountName: "k8sresource",
									Containers: []corev1.Container{{
										Env: []corev1.EnvVar{{
											Name: "TLS_CERT",
											ValueFrom: &corev1.EnvVarSource{
												SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{Name: "secret-name"},
													Key:                  "tls.crt",
												},
											},
										}, {
											Name: "TLS_KEY",
											ValueFrom: &corev1.EnvVarSource{
												SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{Name: "secret-name"},
													Key:                  "tls.key",
												},
											},
										}},
									}},
								},
							},
						},
					},
				},
			},
		},
	}, {
		name: "Valid EventListener with custom resources",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					TriggerRef: "triggerref",
				}},
				Resources: triggersv1beta1.Resources{
					CustomResource: &triggersv1beta1.CustomResource{
						RawExtension: getValidRawData(t),
					},
				},
			},
		},
	}, {
		name: "valid EventListener with embedded TriggerTemplate",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       triggersv1beta1.NamespacedTriggerBindingKind,
						APIVersion: "v1beta1",
					}},
					Template: &triggersv1beta1.EventListenerTemplate{
						Spec: &triggersv1beta1.TriggerTemplateSpec{
							Params: []triggersv1beta1.ParamSpec{{
								Name:        "foo",
								Description: "desc",
								Default:     ptr.String("val"),
							}},
							ResourceTemplates: []triggersv1beta1.TriggerResourceTemplate{{
								RawExtension: test.RawExtension(t, pipelinev1alpha1.PipelineRun{
									TypeMeta: metav1.TypeMeta{
										APIVersion: "tekton.dev/v1beta1",
										Kind:       "TaskRun",
									},
									ObjectMeta: metav1.ObjectMeta{
										Name: "$(tt.params.foo)",
									},
								}),
							}},
						},
					},
				}},
			},
		},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.el.Validate(context.Background())
			if err != nil {
				t.Errorf("EventListener.Validate() expected no error, but got one, EventListener: %v, error: %v", tc.el, err)
			}
		})
	}
}

func TestEventListenerValidate_error(t *testing.T) {
	tests := []struct {
		name string
		el   *triggersv1beta1.EventListener
	}{{
		name: "Invalid EventListener name",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "longlonglonglonglonglonglonglonglonglonglonglonglonglonglname",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with empty TriggerTemplate name",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String(""),
					},
				}},
			},
		},
	}, {
		name: "TriggerBinding with no ref or spec",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{
						APIVersion: "triggers.tekton.dev/triggersv1beta1",
					}},
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "Bindings invalid ref",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Ref: "", Kind: triggersv1beta1.NamespacedTriggerBindingKind}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
				}},
			},
		},
	}, {
		name: "Bindings missing kind",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Ref: "tb", Kind: ""}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
				}},
			},
		},
	}, {
		name: "Template with wrong apiVersion",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt"), APIVersion: "invalid"},
				}},
			},
		},
	}, {
		name: "Template with missing name",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String(""), APIVersion: "triggersv1beta1"},
				}},
			},
		},
	}, {
		name: "Interceptor Name only",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Webhook: &triggersv1beta1.WebhookInterceptor{
							ObjectRef: &corev1.ObjectReference{Name: "svc"},
						},
					}},
				}},
			},
		},
	}, {
		name: "Interceptor Missing ObjectRef",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings:     []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template:     &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*triggersv1beta1.EventInterceptor{{}},
				}},
			},
		},
	}, {
		name: "Interceptor Empty ObjectRef",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Webhook: &triggersv1beta1.WebhookInterceptor{
							ObjectRef: &corev1.ObjectReference{
								Name: "",
							},
						},
					}},
				}},
			},
		},
	}, {
		name: "Valid EventListener with invalid TriggerBinding",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: "badBindingKind", Ref: "tb"}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String(""), APIVersion: "triggersv1beta1"},
				}},
			},
		},
	}, {
		name: "Interceptor Wrong APIVersion",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Webhook: &triggersv1beta1.WebhookInterceptor{
							ObjectRef: &corev1.ObjectReference{
								Name:       "foo",
								APIVersion: "v3",
								Kind:       "Service",
								Namespace:  "default",
							},
						},
					}},
				}},
			},
		},
	}, {
		name: "Interceptor Wrong Kind",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Webhook: &triggersv1beta1.WebhookInterceptor{
							ObjectRef: &corev1.ObjectReference{
								Name:       "foo",
								APIVersion: "v1",
								Kind:       "Deployment",
								Namespace:  "default",
							},
						},
					}},
				}},
			},
		},
	}, {
		name: "Interceptor Non-Canonical Header",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Webhook: &triggersv1beta1.WebhookInterceptor{
							Header: []pipelinev1beta1.Param{{
								Name: "non-canonical-header-key",
								Value: pipelinev1beta1.ArrayOrString{
									Type:      pipelinev1beta1.ParamTypeString,
									StringVal: "valid value",
								},
							}},
							ObjectRef: &corev1.ObjectReference{
								Name:       "foo",
								APIVersion: "v1",
								Kind:       "Service",
								Namespace:  "default",
							},
						},
					}},
				}},
			},
		},
	}, {
		name: "Interceptor Empty Header Name",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Webhook: &triggersv1beta1.WebhookInterceptor{
							Header: []pipelinev1beta1.Param{{
								Name: "",
								Value: pipelinev1beta1.ArrayOrString{
									Type:      pipelinev1beta1.ParamTypeString,
									StringVal: "valid value",
								},
							}},
							ObjectRef: &corev1.ObjectReference{
								Name:       "foo",
								APIVersion: "v1",
								Kind:       "Service",
								Namespace:  "default",
							},
						},
					}},
				}},
			},
		},
	}, {
		name: "Interceptor Empty Header Value",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Webhook: &triggersv1beta1.WebhookInterceptor{
							Header: []pipelinev1beta1.Param{{
								Name: "Valid-Header-Key",
								Value: pipelinev1beta1.ArrayOrString{
									Type:      pipelinev1beta1.ParamTypeString,
									StringVal: "",
								},
							}},
							ObjectRef: &corev1.ObjectReference{
								Name:       "foo",
								APIVersion: "v1",
								Kind:       "Service",
								Namespace:  "default",
							},
						},
					}},
				}},
			},
		},
	}, {
		name: "Multiple interceptors set",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						DeprecatedGitHub:    &triggersv1beta1.GitHubInterceptor{},
						DeprecatedGitLab:    &triggersv1beta1.GitLabInterceptor{},
						DeprecatedBitbucket: &triggersv1beta1.BitbucketInterceptor{},
					}},
				}},
			},
		},
	}, {
		name: "CEL interceptor with no filter or overlays",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						DeprecatedCEL: &triggersv1beta1.CELInterceptor{},
					}},
				}},
			},
		},
	}, {
		name: "CEL interceptor with bad filter expression",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						DeprecatedCEL: &triggersv1beta1.CELInterceptor{
							Filter: "body.value == 'test'",
						},
					}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String(""), APIVersion: "triggersv1beta1"},
				}},
			},
		},
	}, {
		name: "CEL interceptor with bad overlay expression",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						DeprecatedCEL: &triggersv1beta1.CELInterceptor{
							Overlays: []triggersv1beta1.CELOverlay{{
								Key:        "value",
								Expression: "'testing')",
							}},
						},
					}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String(""), APIVersion: "triggersv1beta1"},
				}},
			},
		},
	}, {
		name: "Triggers name has invalid label characters",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Name:     "github.com/tektoncd/triggers",
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String(""), APIVersion: "triggersv1beta1"},
				}},
			},
		},
	}, {
		name: "Triggers name is longer than the allowable label value (63 characters)",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Name:     "1234567890123456789012345678901234567890123456789012345678901234",
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String(""), APIVersion: "triggersv1beta1"},
				}},
			},
		},
	}, {
		name: "user specify invalid deprecated replicas",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
				DeprecatedReplicas: ptr.Int32(-1),
				Resources: triggersv1beta1.Resources{
					KubernetesResource: &triggersv1beta1.KubernetesResource{
						Replicas: ptr.Int32(-1),
					},
				},
			},
		},
	}, {
		name: "user specify invalid replicas",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
				Resources: triggersv1beta1.Resources{
					KubernetesResource: &triggersv1beta1.KubernetesResource{
						Replicas: ptr.Int32(-1),
					},
				},
			},
		},
	}, {
		name: "user specify multiple containers",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Resources: triggersv1beta1.Resources{
					KubernetesResource: &triggersv1beta1.KubernetesResource{
						WithPodSpec: duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Env: []corev1.EnvVar{{Name: "HTTP"}},
									}, {
										Env: []corev1.EnvVar{{Name: "TCP"}},
									}},
								},
							},
						},
					},
				},
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "user specifies an unsupported podspec field",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Resources: triggersv1beta1.Resources{
					KubernetesResource: &triggersv1beta1.KubernetesResource{
						WithPodSpec: duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								Spec: corev1.PodSpec{
									NodeName: "minikube",
								},
							},
						},
					},
				},
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "user specifies an unsupported container fields",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Resources: triggersv1beta1.Resources{
					KubernetesResource: &triggersv1beta1.KubernetesResource{
						WithPodSpec: duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Name: "containername",
										Env: []corev1.EnvVar{{
											Name:  "key",
											Value: "value",
										}, {
											Name: "key1",
											ValueFrom: &corev1.EnvVarSource{
												ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
													Key: "key",
												},
											},
										}},
									}},
								},
							},
						},
					},
				},
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "user specifies an invalid env for TLS connection",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Resources: triggersv1beta1.Resources{
					KubernetesResource: &triggersv1beta1.KubernetesResource{
						WithPodSpec: duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Env: []corev1.EnvVar{{
											Name: "TLS_CERT",
											ValueFrom: &corev1.EnvVarSource{
												SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{
														Name: "secret-name",
													},
													Key: "tls.key",
												},
											},
										}},
									}},
								},
							},
						},
					},
				},
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "user specify both kubernetes and custom resources",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       "TriggerBinding",
						APIVersion: "triggersv1beta1",
					}},
				}},
				Resources: triggersv1beta1.Resources{
					KubernetesResource: &triggersv1beta1.KubernetesResource{
						ServiceType: "NodePort",
					},
					CustomResource: &triggersv1beta1.CustomResource{
						RawExtension: runtime.RawExtension{Raw: []byte(`{"rt1": "value"}`)},
					},
				},
			},
		},
	}, {
		name: "user specify multiple containers, unsupported podspec and container field in custom resources",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       "TriggerBinding",
						APIVersion: "triggersv1beta1",
					}},
				}},
				Resources: triggersv1beta1.Resources{
					CustomResource: &triggersv1beta1.CustomResource{
						RawExtension: getRawData(t),
					},
				},
			},
		},
	}, {
		name: "specify TriggerTemplate along with TriggerRef",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
					TriggerRef: "triggerref",
				}},
				Resources: triggersv1beta1.Resources{
					CustomResource: &triggersv1beta1.CustomResource{
						RawExtension: getValidRawData(t),
					},
				},
			},
		},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.el.Validate(context.Background()); err == nil {
				t.Errorf("EventListener.Validate() expected error, but get none, EventListener: %v", tc.el)
			}
		})
	}
}

func getRawData(t *testing.T) runtime.RawExtension {
	return test.RawExtension(t, duckv1.WithPod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "serving.knative.dev/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "knativeservice",
		},
		Spec: duckv1.WithPodSpec{Template: duckv1.PodSpecable{
			Spec: corev1.PodSpec{
				ServiceAccountName: "tekton-triggers-example-sa",
				NodeName:           "minikube",
				Containers: []corev1.Container{{
					Name: "first-container",
				}, {
					Env: []corev1.EnvVar{{
						Name: "key",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "test"},
								Key:                  "a.crt",
							},
						},
					}},
				}},
			},
		}},
	})
}

func getValidRawData(t *testing.T) runtime.RawExtension {
	return test.RawExtension(t, duckv1.WithPod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "serving.knative.dev/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "knativeservice",
		},
		Spec: duckv1.WithPodSpec{Template: duckv1.PodSpecable{
			Spec: corev1.PodSpec{
				ServiceAccountName: "tekton-triggers-example-sa",
				Containers: []corev1.Container{{
					Env: []corev1.EnvVar{{
						Name: "key",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "test"},
								Key:                  "a.crt",
							},
						},
					}},
				}},
			},
		}},
	})
}
