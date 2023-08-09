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
	"strings"
	"testing"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/pkg/apis/triggers"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
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
	el := &v1alpha1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.Repeat("foo", 64), // Length should be lower than 63
			Namespace: "namespace",
		},
		Spec: v1alpha1.EventListenerSpec{
			Triggers: []v1alpha1.EventListenerTrigger{{
				Template: &v1alpha1.EventListenerTemplate{
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
		el   *v1alpha1.EventListener
	}{{
		name: "TriggerTemplate Does Not Exist",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Template: &v1alpha1.EventListenerTemplate{
						Ref:        ptr.String("dne"),
						APIVersion: "v1alpha1",
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener No TriggerBinding",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Template: &v1alpha1.EventListenerTemplate{
						Ref:        ptr.String("tt"),
						APIVersion: "v1alpha1",
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with TriggerRef",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					TriggerRef: "tt",
				}},
			},
		},
	}, {
		name: "Valid EventListener with Annotation",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "name",
				Namespace:   "namespace",
				Annotations: map[string]string{triggers.PayloadValidationAnnotation: "true"},
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					TriggerRef: "tt",
				}},
			},
		},
	}, {
		name: "Valid EventListener with TriggerBinding",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						APIVersion: "v1alpha1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with ClusterTriggerBinding",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1alpha1.ClusterTriggerBindingKind,
						APIVersion: "v1alpha1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with multiple TriggerBindings",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{
						Ref:        "tb1",
						Kind:       v1alpha1.ClusterTriggerBindingKind,
						APIVersion: "v1alpha1", // TODO: APIVersions seem wrong?
					}, {
						Ref:        "tb2",
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						APIVersion: "v1alpha1", // TODO: APIVersions seem wrong?
					}, {
						Ref:        "tb3",
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						APIVersion: "v1alpha1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener Webhook Interceptor",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Interceptors: []*v1alpha1.EventInterceptor{{
						Webhook: &v1alpha1.WebhookInterceptor{
							ObjectRef: &corev1.ObjectReference{
								Name:       "svc",
								Kind:       "Service",
								APIVersion: "v1",
								Namespace:  "namespace",
							},
						},
					}},
					Bindings: []*v1alpha1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						APIVersion: "v1alpha1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener Interceptor With Header",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Interceptors: []*v1alpha1.EventInterceptor{{
						Webhook: &v1alpha1.WebhookInterceptor{
							Header: []pipelinev1beta1.Param{{
								Name: "Valid-Header-Key",
								Value: pipelinev1beta1.ParamValue{
									Type:      pipelinev1beta1.ParamTypeString,
									StringVal: "valid-value",
								},
							}, {
								Name: "Valid-Header-Key2",
								Value: pipelinev1beta1.ParamValue{
									Type:      pipelinev1beta1.ParamTypeString,
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
					Bindings: []*v1alpha1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						APIVersion: "v1alpha1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener Two Triggers",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						APIVersion: "v1alpha1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}, {
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
					Bindings: []*v1alpha1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						APIVersion: "v1alpha1", // TODO: APIVersions seem wrong?
					}},
				}},
			},
		},
	}, {
		name: "Valid EventListener with CEL interceptor",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Interceptors: []*v1alpha1.EventInterceptor{{
						Ref: v1alpha1.InterceptorRef{
							Name:       "cel",
							Kind:       v1alpha1.ClusterInterceptorKind,
							APIVersion: "triggers.tekton.dev/v1alpha1",
						},
						Params: []v1alpha1.InterceptorParams{{
							Name:  "filter",
							Value: test.ToV1JSON(t, "body.value == test"),
						}, {
							Name: "overlays",
							Value: test.ToV1JSON(t, []v1alpha1.CELOverlay{{
								Key:        "value",
								Expression: "testing",
							}}),
						}},
					}},
					Bindings: []*v1alpha1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						APIVersion: "v1alpha1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with no trigger name",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						APIVersion: "v1alpha1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "Namespace selector with label selector",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				NamespaceSelector: v1alpha1.NamespaceSelector{
					MatchNames: []string{"foo"},
				},
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"foo": "bar",
					},
				},
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Ref: "bindingRef", Kind: v1alpha1.NamespacedTriggerBindingKind}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String("tt")},
				}},
			},
		},
	}, {
		name: "Valid EventListener with kubernetes env for podspec",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
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
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
						Replicas: ptr.Int32(1),
					},
				},
			},
		},
	}, {
		name: "Valid EventListener with env for TLS connection",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
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
		name: "Valid EventListener with probes",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
						WithPodSpec: duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								Spec: corev1.PodSpec{
									ServiceAccountName: "k8sresource",
									Containers: []corev1.Container{{
										ReadinessProbe: &corev1.Probe{
											InitialDelaySeconds: int32(10),
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
		name: "Valid EventListener with custom resources",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					TriggerRef: "triggerref",
				}},
				Resources: v1alpha1.Resources{
					CustomResource: &v1alpha1.CustomResource{
						RawExtension: getValidRawData(t),
					},
				},
			},
		},
	}, {
		name: "valid EventListener with embedded TriggerTemplate",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1alpha1.NamespacedTriggerBindingKind,
						APIVersion: "v1alpha1",
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Spec: &v1alpha1.TriggerTemplateSpec{
							Params: []v1alpha1.ParamSpec{{
								Name:        "foo",
								Description: "desc",
								Default:     ptr.String("val"),
							}},
							ResourceTemplates: []v1alpha1.TriggerResourceTemplate{{
								RawExtension: test.RawExtension(t, pipelinev1.PipelineRun{
									TypeMeta: metav1.TypeMeta{
										APIVersion: "tekton.dev/v1",
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
		el   *v1alpha1.EventListener
	}{{
		name: "Invalid EventListener name",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "longlonglonglonglonglonglonglonglonglonglonglonglonglonglname",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with empty TriggerTemplate name",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String(""),
					},
				}},
			},
		},
	}, {
		name: "Invalid Annotation value",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "name",
				Namespace:   "namespace",
				Annotations: map[string]string{triggers.PayloadValidationAnnotation: "xyz"},
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					TriggerRef: "tt",
				}},
			},
		},
	}, {
		name: "TriggerBinding with no ref or spec",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{
						APIVersion: "triggers.tekton.dev/v1alpha1",
					}},
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "Bindings invalid ref",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Ref: "", Kind: v1alpha1.NamespacedTriggerBindingKind}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String("tt")},
				}},
			},
		},
	}, {
		name: "Bindings missing kind",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Ref: "tb", Kind: ""}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String("tt")},
				}},
			},
		},
	}, {
		name: "Template with wrong apiVersion",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String("tt"), APIVersion: "invalid"},
				}},
			},
		},
	}, {
		name: "Template with missing name",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String(""), APIVersion: "v1alpha1"},
				}},
			},
		},
	}, {
		name: "Interceptor Name only",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*v1alpha1.EventInterceptor{{
						Webhook: &v1alpha1.WebhookInterceptor{
							ObjectRef: &corev1.ObjectReference{Name: "svc"},
						},
					}},
				}},
			},
		},
	}, {
		name: "Interceptor Missing ObjectRef",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings:     []*v1alpha1.EventListenerBinding{{Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template:     &v1alpha1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*v1alpha1.EventInterceptor{{}},
				}},
			},
		},
	}, {
		name: "Interceptor Empty ObjectRef",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*v1alpha1.EventInterceptor{{
						Webhook: &v1alpha1.WebhookInterceptor{
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
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Kind: "badBindingKind", Ref: "tb"}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String(""), APIVersion: "v1alpha1"},
				}},
			},
		},
	}, {
		name: "Interceptor Wrong APIVersion",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*v1alpha1.EventInterceptor{{
						Webhook: &v1alpha1.WebhookInterceptor{
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
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*v1alpha1.EventInterceptor{{
						Webhook: &v1alpha1.WebhookInterceptor{
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
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*v1alpha1.EventInterceptor{{
						Webhook: &v1alpha1.WebhookInterceptor{
							Header: []pipelinev1beta1.Param{{
								Name: "non-canonical-header-key",
								Value: pipelinev1beta1.ParamValue{
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
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*v1alpha1.EventInterceptor{{
						Webhook: &v1alpha1.WebhookInterceptor{
							Header: []pipelinev1beta1.Param{{
								Name: "",
								Value: pipelinev1beta1.ParamValue{
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
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*v1alpha1.EventInterceptor{{
						Webhook: &v1alpha1.WebhookInterceptor{
							Header: []pipelinev1beta1.Param{{
								Name: "Valid-Header-Key",
								Value: pipelinev1beta1.ParamValue{
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
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*v1alpha1.EventInterceptor{{
						DeprecatedGitHub:    &v1alpha1.GitHubInterceptor{},
						DeprecatedGitLab:    &v1alpha1.GitLabInterceptor{},
						DeprecatedBitbucket: &v1alpha1.BitbucketInterceptor{},
					}},
				}},
			},
		},
	}, {
		name: "CEL interceptor with no filter or overlays",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String("tt")},
					Interceptors: []*v1alpha1.EventInterceptor{{
						DeprecatedCEL: &v1alpha1.CELInterceptor{},
					}},
				}},
			},
		},
	}, {
		name: "CEL interceptor with bad filter expression",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Interceptors: []*v1alpha1.EventInterceptor{{
						DeprecatedCEL: &v1alpha1.CELInterceptor{
							Filter: "body.value == 'test'",
						},
					}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String(""), APIVersion: "v1alpha1"},
				}},
			},
		},
	}, {
		name: "CEL interceptor with bad overlay expression",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Interceptors: []*v1alpha1.EventInterceptor{{
						DeprecatedCEL: &v1alpha1.CELInterceptor{
							Overlays: []v1alpha1.CELOverlay{{
								Key:        "value",
								Expression: "'testing')",
							}},
						},
					}},
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String(""), APIVersion: "v1alpha1"},
				}},
			},
		},
	}, {
		name: "Triggers name has invalid label characters",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Name:     "github.com/tektoncd/triggers",
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String(""), APIVersion: "v1alpha1"},
				}},
			},
		},
	}, {
		name: "Triggers name is longer than the allowable label value (63 characters)",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Name:     "1234567890123456789012345678901234567890123456789012345678901234",
					Template: &v1alpha1.EventListenerTemplate{Ref: ptr.String(""), APIVersion: "v1alpha1"},
				}},
			},
		},
	}, {
		name: "user specify invalid replicas",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
						Replicas: ptr.Int32(-1),
					},
				},
			},
		},
	}, {
		name: "user specify multiple containers",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
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
				Triggers: []v1alpha1.EventListenerTrigger{{
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "user specifies an unsupported podspec field",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
						WithPodSpec: duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								Spec: corev1.PodSpec{
									NodeName: "minikube",
								},
							},
						},
					},
				},
				Triggers: []v1alpha1.EventListenerTrigger{{
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "user specifies an unsupported container fields",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
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
				Triggers: []v1alpha1.EventListenerTrigger{{
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "user specifies an invalid env for TLS connection",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
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
				Triggers: []v1alpha1.EventListenerTrigger{{
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "user specify both kubernetes and custom resources",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       "TriggerBinding",
						APIVersion: "v1alpha1",
					}},
				}},
				Resources: v1alpha1.Resources{
					KubernetesResource: &v1alpha1.KubernetesResource{
						ServiceType: "NodePort",
					},
					CustomResource: &v1alpha1.CustomResource{
						RawExtension: runtime.RawExtension{Raw: []byte(`{"rt1": "value"}`)},
					},
				},
			},
		},
	}, {
		name: "user specify multiple containers, unsupported podspec and container field in custom resources",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Bindings: []*v1alpha1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       "TriggerBinding",
						APIVersion: "v1alpha1",
					}},
				}},
				Resources: v1alpha1.Resources{
					CustomResource: &v1alpha1.CustomResource{
						RawExtension: getRawData(t),
					},
				},
			},
		},
	}, {
		name: "specify TriggerTemplate along with TriggerRef",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Template: &v1alpha1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
					TriggerRef: "triggerref",
				}},
				Resources: v1alpha1.Resources{
					CustomResource: &v1alpha1.CustomResource{
						RawExtension: getValidRawData(t),
					},
				},
			},
		},
	}, {
		name: "custom resource with probe data",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					TriggerRef: "triggerref",
				}},
				Resources: v1alpha1.Resources{
					CustomResource: &v1alpha1.CustomResource{
						RawExtension: getRawDataWithPobes(t),
					},
				},
			},
		},
	}, {
		name: "empty spec for eventlistener",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
		},
	}, {
		name: "invalid interceptor for eventlistener",
		el: &v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{{
					Name: "test",
					Interceptors: []*v1alpha1.EventInterceptor{{
						Ref: v1alpha1.InterceptorRef{
							Name:       "cel",
							Kind:       v1alpha1.ClusterInterceptorKind,
							APIVersion: "triggers.tekton.dev/v1alpha1",
						},
					},
						nil,
					},
				}},
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

func getRawDataWithPobes(t *testing.T) runtime.RawExtension {
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
					ReadinessProbe: &corev1.Probe{
						FailureThreshold: int32(10),
					},
					LivenessProbe: &corev1.Probe{
						FailureThreshold: int32(10),
					},
				}},
			},
		}},
	})
}
