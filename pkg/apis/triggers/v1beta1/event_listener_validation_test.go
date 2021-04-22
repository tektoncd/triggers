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

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/ptr"
)

func Test_EventListenerValidate(t *testing.T) {
	tests := []struct {
		name string
		el   *v1beta1.EventListener
	}{{
		name: "TriggerTemplate Does Not Exist",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Template: &v1beta1.EventListenerTemplate{
						Ref:        ptr.String("dne"),
						APIVersion: "v1beta1", // TODO: test with v1alpha1 version as well.
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener No TriggerBinding",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Template: &v1beta1.EventListenerTemplate{
						Ref:        ptr.String("tt"),
						APIVersion: "v1beta1", // TODO: test with v1alpha1 version as well.
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with TriggerRef",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					TriggerRef: "tt",
				}},
			},
		},
	}, {
		name: "Valid EventListener with TriggerBinding",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1beta1.NamespacedTriggerBindingKind,
						APIVersion: "v1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with ClusterTriggerBinding",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1beta1.ClusterTriggerBindingKind,
						APIVersion: "v1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with multiple TriggerBindings",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{{
						Ref:        "tb1",
						Kind:       v1beta1.ClusterTriggerBindingKind,
						APIVersion: "v1beta1", // TODO: APIVersions seem wrong?
					}, {
						Ref:        "tb2",
						Kind:       v1beta1.NamespacedTriggerBindingKind,
						APIVersion: "v1beta1", // TODO: APIVersions seem wrong?
					}, {
						Ref:        "tb3",
						Kind:       v1beta1.NamespacedTriggerBindingKind,
						APIVersion: "v1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener Webhook Interceptor",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Interceptors: []*v1beta1.EventInterceptor{{
						Webhook: &v1beta1.WebhookInterceptor{
							ObjectRef: &corev1.ObjectReference{
								Name:       "svc",
								Kind:       "Service",
								APIVersion: "v1",
								Namespace:  "namespace",
							},
						},
					}},
					Bindings: []*v1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1beta1.NamespacedTriggerBindingKind,
						APIVersion: "v1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener Interceptor With Header",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Interceptors: []*v1beta1.EventInterceptor{{
						Webhook: &v1beta1.WebhookInterceptor{
							Header: []pipelinev1.Param{{
								Name: "Valid-Header-Key",
								Value: pipelinev1.ArrayOrString{
									Type:      pipelinev1.ParamTypeString,
									StringVal: "valid-value",
								},
							}, {
								Name: "Valid-Header-Key2",
								Value: pipelinev1.ArrayOrString{
									Type:      pipelinev1.ParamTypeString,
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
					Bindings: []*v1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1beta1.NamespacedTriggerBindingKind,
						APIVersion: "v1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener Two Triggers",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1beta1.NamespacedTriggerBindingKind,
						APIVersion: "v1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}, {
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt2"),
					},
					Bindings: []*v1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1beta1.NamespacedTriggerBindingKind,
						APIVersion: "v1beta1", // TODO: APIVersions seem wrong?
					}},
				}},
			},
		},
	}, {
		name: "Valid EventListener with CEL interceptor",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Interceptors: []*v1beta1.EventInterceptor{{
						Ref: v1beta1.InterceptorRef{
							Name:       "cel",
							Kind:       v1beta1.ClusterInterceptorKind,
							APIVersion: "triggers.tekton.dev/v1beta1",
						},
						Params: []v1beta1.InterceptorParams{{
							Name:  "filter",
							Value: test.ToV1JSON(t, "body.value == test"),
						}, {
							Name: "overlays",
							Value: test.ToV1JSON(t, []v1beta1.CELOverlay{{
								Key:        "value",
								Expression: "testing",
							}}),
						}},
					}},
					Bindings: []*v1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1beta1.NamespacedTriggerBindingKind,
						APIVersion: "v1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with no trigger name",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       v1beta1.NamespacedTriggerBindingKind,
						APIVersion: "v1beta1", // TODO: APIVersions seem wrong?
					}},
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "Namespace selector with label selector",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				NamespaceSelector: v1beta1.NamespaceSelector{
					MatchNames: []string{"foo"},
				},
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"foo": "bar",
					},
				},
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{{Ref: "bindingRef", Kind: v1beta1.NamespacedTriggerBindingKind}},
					Template: &v1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
				}},
			},
		},
	}, {
		name: "Valid EventListener with kubernetes env for podspec",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
				Resources: v1beta1.Resources{
					KubernetesResource: &v1beta1.KubernetesResource{
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
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Resources: v1beta1.Resources{
					KubernetesResource: &v1beta1.KubernetesResource{
						Replicas: ptr.Int32(1),
					},
				},
			},
		},
	}, {
		name: "Valid EventListener with env for TLS connection",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
				Resources: v1beta1.Resources{
					KubernetesResource: &v1beta1.KubernetesResource{
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
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       "TriggerBinding",
						APIVersion: "v1beta1",
					}},
					TriggerRef: "triggerref",
				}},
				Resources: v1beta1.Resources{
					CustomResource: &v1beta1.CustomResource{
						RawExtension: getValidRawData(t),
					},
				},
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
		el   *v1beta1.EventListener
	}{{
		name: "Invalid EventListener name",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "longlonglonglonglonglonglonglonglonglonglonglonglonglonglname",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "Valid EventListener with empty TriggerTemplate name",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String(""),
					},
				}},
			},
		},
	}, {
		name: "TriggerBinding with no ref or spec",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{{
						APIVersion: "triggers.tekton.dev/v1beta1",
					}},
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "Bindings invalid ref",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{{Ref: "", Kind: v1beta1.NamespacedTriggerBindingKind}},
					Template: &v1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
				}},
			},
		},
	}, {
		name: "Bindings missing kind",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{{Ref: "tb", Kind: ""}},
					Template: &v1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
				}},
			},
		},
	}, {
		name: "Template with wrong apiVersion",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{{Kind: v1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &v1beta1.EventListenerTemplate{Ref: ptr.String("tt"), APIVersion: "invalid"},
				}},
			},
		},
	}, {
		name: "Template with missing name",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{{Kind: v1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
					Template: &v1beta1.EventListenerTemplate{Ref: ptr.String(""), APIVersion: "v1beta1"},
				}},
			},
		},
	}, {
		//	name: "Interceptor Name only",
		//	el: bldr.EventListener("name", "namespace",
		//		bldr.EventListenerSpec(
		//			bldr.EventListenerTrigger("tt", "v1beta1",
		//				bldr.EventListenerTriggerBinding("tb", "", "v1beta1"),
		//				bldr.EventListenerTriggerInterceptor("svc", "", "", ""),
		//			))),
		//}, {
		//	name: "Interceptor Missing ObjectRef",
		//	el: &v1beta1.EventListener{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name:      "name",
		//			Namespace: "namespace",
		//		},
		//		Spec: v1beta1.EventListenerSpec{
		//			Triggers: []v1beta1.EventListenerTrigger{{
		//				Bindings:     []*v1beta1.EventListenerBinding{{Kind: v1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
		//				Template:     &v1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
		//				Interceptors: []*v1beta1.EventInterceptor{{}},
		//			}},
		//		},
		//	},
		//}, {
		//	name: "Interceptor Empty ObjectRef",
		//	el: &v1beta1.EventListener{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name:      "name",
		//			Namespace: "namespace",
		//		},
		//		Spec: v1beta1.EventListenerSpec{
		//			Triggers: []v1beta1.EventListenerTrigger{{
		//				Bindings: []*v1beta1.EventListenerBinding{{Kind: v1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
		//				Template: &v1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
		//				Interceptors: []*v1beta1.EventInterceptor{{
		//					Webhook: &v1beta1.WebhookInterceptor{
		//						ObjectRef: &corev1.ObjectReference{
		//							Name: "",
		//						},
		//					},
		//				}},
		//			}},
		//		},
		//	},
		//}, {
		//	name: "Valid EventListener with invalid TriggerBinding",
		//	el: bldr.EventListener("name", "namespace",
		//		bldr.EventListenerSpec(
		//			bldr.EventListenerTrigger("tt", "v1beta1",
		//				bldr.EventListenerTriggerBinding("tb", "NamespaceTriggerBinding", "v1beta1"),
		//			))),
		//}, {
		//	name: "Interceptor Wrong APIVersion",
		//	el: bldr.EventListener("name", "namespace",
		//		bldr.EventListenerSpec(
		//			bldr.EventListenerTrigger("tt", "v1beta1",
		//				bldr.EventListenerTriggerBinding("tb", "", "v1beta1"),
		//				bldr.EventListenerTriggerInterceptor("foo", "v3", "Service", ""),
		//			))),
		//}, {
		//	name: "Interceptor Wrong Kind",
		//	el: bldr.EventListener("name", "namespace",
		//		bldr.EventListenerSpec(
		//			bldr.EventListenerTrigger("tt", "v1beta1",
		//				bldr.EventListenerTriggerBinding("tb", "", "v1beta1"),
		//				bldr.EventListenerTriggerInterceptor("foo", "v1", "Deployment", ""),
		//			))),
		//}, {
		//	name: "Interceptor Non-Canonical Header",
		//	el: bldr.EventListener("name", "namespace",
		//		bldr.EventListenerSpec(
		//			bldr.EventListenerTrigger("tt", "v1beta1",
		//				bldr.EventListenerTriggerBinding("tb", "", "v1beta1"),
		//				bldr.EventListenerTriggerInterceptor("foo", "v1", "Deployment", "",
		//					bldr.EventInterceptorParam("non-canonical-header-key", "valid value"),
		//				),
		//			))),
		//}, {
		//	name: "Interceptor Empty Header Name",
		//	el: bldr.EventListener("name", "namespace",
		//		bldr.EventListenerSpec(
		//			bldr.EventListenerTrigger("tt", "v1beta1",
		//				bldr.EventListenerTriggerBinding("tb", "", "v1beta1"),
		//				bldr.EventListenerTriggerInterceptor("foo", "v1", "Deployment", "",
		//					bldr.EventInterceptorParam("", "valid value"),
		//				),
		//			))),
		//}, {
		//	name: "Interceptor Empty Header Value",
		//	el: bldr.EventListener("name", "namespace",
		//		bldr.EventListenerSpec(
		//			bldr.EventListenerTrigger("tt", "v1beta1",
		//				bldr.EventListenerTriggerBinding("tb", "", "v1beta1"),
		//				bldr.EventListenerTriggerInterceptor("foo", "v1", "Deployment", "",
		//					bldr.EventInterceptorParam("Valid-Header-Key", ""),
		//				),
		//			))),
		//}, {
		//	name: "Multiple interceptors set",
		//	el: &v1beta1.EventListener{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name:      "name",
		//			Namespace: "namespace",
		//		},
		//		Spec: v1beta1.EventListenerSpec{
		//			Triggers: []v1beta1.EventListenerTrigger{{
		//				Bindings: []*v1beta1.EventListenerBinding{{Kind: v1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
		//				Template: &v1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
		//				Interceptors: []*v1beta1.EventInterceptor{{
		//					DeprecatedGitHub:    &v1beta1.GitHubInterceptor{},
		//					DeprecatedGitLab:    &v1beta1.GitLabInterceptor{},
		//					DeprecatedBitbucket: &v1beta1.BitbucketInterceptor{},
		//				}},
		//			}},
		//		},
		//	},
		//}, {
		//	name: "CEL interceptor with no filter or overlays",
		//	el: &v1beta1.EventListener{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name:      "name",
		//			Namespace: "namespace",
		//		},
		//		Spec: v1beta1.EventListenerSpec{
		//			Triggers: []v1beta1.EventListenerTrigger{{
		//				Bindings: []*v1beta1.EventListenerBinding{{Kind: v1beta1.NamespacedTriggerBindingKind, Ref: "tb"}},
		//				Template: &v1beta1.EventListenerTemplate{Ref: ptr.String("tt")},
		//				Interceptors: []*v1beta1.EventInterceptor{{
		//					DeprecatedCEL: &v1beta1.CELInterceptor{},
		//				}},
		//			}},
		//		},
		//	},
		//}, {
		//	name: "CEL interceptor with bad filter expression",
		//	el: bldr.EventListener("name", "namespace",
		//		bldr.EventListenerSpec(
		//			bldr.EventListenerTrigger("tt", "v1beta1",
		//				bldr.EventListenerTriggerBinding("tb", "", "v1beta1"),
		//				bldr.EventListenerCELInterceptor("body.value == 'test')"),
		//			))),
		//}, {
		//	name: "CEL interceptor with bad overlay expression",
		//	el: bldr.EventListener("name", "namespace",
		//		bldr.EventListenerSpec(
		//			bldr.EventListenerTrigger("tt", "v1beta1",
		//				bldr.EventListenerTriggerBinding("tb", "", "v1beta1"),
		//				bldr.EventListenerCELInterceptor("", bldr.EventListenerCELOverlay("body.value", "'testing')")),
		//			))),
		//}, {
		//	name: "Triggers name has invalid label characters",
		//	el: bldr.EventListener("name", "namespace",
		//		bldr.EventListenerSpec(
		//			bldr.EventListenerTrigger("tt", "v1beta1",
		//				bldr.EventListenerTriggerBinding("tb", "", "v1beta1"),
		//				bldr.EventListenerTriggerName("github.com/tektoncd/triggers"),
		//			))),
		//}, {
		//	name: "Triggers name is longer than the allowable label value (63 characters)",
		//	el: bldr.EventListener("name", "namespace",
		//		bldr.EventListenerSpec(
		//			bldr.EventListenerTrigger("tt", "v1beta1",
		//				bldr.EventListenerTriggerBinding("tb", "", "v1beta1"),
		//				bldr.EventListenerTriggerName("1234567890123456789012345678901234567890123456789012345678901234"),
		//			))),
		//}, {
		name: "user specify invalid deprecated replicas",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
				DeprecatedReplicas: ptr.Int32(-1),
				Resources: v1beta1.Resources{
					KubernetesResource: &v1beta1.KubernetesResource{
						Replicas: ptr.Int32(-1),
					},
				},
			},
		},
	}, {
		name: "user specify invalid replicas",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
				Resources: v1beta1.Resources{
					KubernetesResource: &v1beta1.KubernetesResource{
						Replicas: ptr.Int32(-1),
					},
				},
			},
		},
	}, {
		name: "user specify multiple containers",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Resources: v1beta1.Resources{
					KubernetesResource: &v1beta1.KubernetesResource{
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
				Triggers: []v1beta1.EventListenerTrigger{{
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "user specifies an unsupported podspec field",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Resources: v1beta1.Resources{
					KubernetesResource: &v1beta1.KubernetesResource{
						WithPodSpec: duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								Spec: corev1.PodSpec{
									NodeName: "minikube",
								},
							},
						},
					},
				},
				Triggers: []v1beta1.EventListenerTrigger{{
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "user specifies an unsupported container fields",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Resources: v1beta1.Resources{
					KubernetesResource: &v1beta1.KubernetesResource{
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
				Triggers: []v1beta1.EventListenerTrigger{{
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "user specifies an invalid env for TLS connection",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Resources: v1beta1.Resources{
					KubernetesResource: &v1beta1.KubernetesResource{
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
				Triggers: []v1beta1.EventListenerTrigger{{
					Template: &v1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
			},
		},
	}, {
		name: "user specify both kubernetes and custom resources",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       "TriggerBinding",
						APIVersion: "v1beta1",
					}},
				}},
				Resources: v1beta1.Resources{
					KubernetesResource: &v1beta1.KubernetesResource{
						ServiceType: "NodePort",
					},
					CustomResource: &v1beta1.CustomResource{
						RawExtension: runtime.RawExtension{Raw: []byte(`{"rt1": "value"}`)},
					},
				},
			},
		},
	}, {
		name: "user specify multiple containers, unsupported podspec and container field in custom resources",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.EventListenerSpec{
				Triggers: []v1beta1.EventListenerTrigger{{
					Bindings: []*v1beta1.EventListenerBinding{{
						Ref:        "tb",
						Kind:       "TriggerBinding",
						APIVersion: "v1beta1",
					}},
				}},
				Resources: v1beta1.Resources{
					CustomResource: &v1beta1.CustomResource{
						RawExtension: getRawData(t),
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
