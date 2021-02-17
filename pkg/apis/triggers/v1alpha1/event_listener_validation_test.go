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

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/test"
	bldr "github.com/tektoncd/triggers/test/builder"
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
		el   *v1alpha1.EventListener
	}{{
		name: "TriggerTemplate Does Not Exist",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("dne", "v1alpha1"))),
	}, {
		name: "Valid EventListener No TriggerBinding",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1"))),
	}, {
		name: "Valid EventListener with TriggerRef",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTriggerRef("tt"))),
	}, {
		name: "Valid EventListener with TriggerBinding",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "TriggerBinding", "v1alpha1"),
				))),
	}, {
		name: "Valid EventListener with ClusterTriggerBinding",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "ClusterTriggerBinding", "v1alpha1"),
				))),
	}, {
		name: "Valid EventListener with multiple TriggerBindings",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb1", "ClusterTriggerBinding", "v1alpha1"),
					bldr.EventListenerTriggerBinding("tb2", "TriggerBinding", "v1alpha1"),
					bldr.EventListenerTriggerBinding("tb3", "", "v1alpha1"),
				))),
	}, {
		name: "Valid EventListener No Interceptor",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
				))),
	}, {
		name: "Valid EventListener Interceptor",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerTriggerInterceptor("svc", "v1", "Service", "namespace"),
				))),
	}, {
		name: "Valid EventListener Interceptor With Header",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerTriggerInterceptor("svc", "v1", "Service", "namespace",
						bldr.EventInterceptorParam("Valid-Header-Key", "valid value"),
					),
				))),
	}, {
		name: "Valid EventListener Interceptor With Headers",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerTriggerInterceptor("svc", "v1", "Service", "namespace",
						bldr.EventInterceptorParam("Valid-Header-Key1", "valid value1"),
						bldr.EventInterceptorParam("Valid-Header-Key1", "valid value2"),
						bldr.EventInterceptorParam("Valid-Header-Key2", "valid value"),
					),
				))),
	}, {
		name: "Valid EventListener Two Triggers",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerTriggerInterceptor("svc", "v1", "Service", "namespace"),
				),
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
				))),
	}, {
		name: "Valid EventListener with CEL interceptor",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerCELInterceptor("body.value == 'test'"),
				))),
	}, {
		name: "Valid EventListener with no trigger name",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
				))),
	}, {
		name: "Valid EventListener with CEL overlays",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerCELInterceptor("", bldr.EventListenerCELOverlay("body.value", "'testing'")),
				))),
	}, {
		name: "Valid EventListener with kubernetes env for podspec",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1"),
				bldr.EventListenerResources(
					bldr.EventListenerKubernetesResources(
						bldr.EventListenerPodSpec(duckv1.WithPodSpec{
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
						}),
					)),
			)),
	}, {
		name: "Valid EventListener with env for TLS connection",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1"),
				bldr.EventListenerResources(
					bldr.EventListenerKubernetesResources(
						bldr.EventListenerPodSpec(duckv1.WithPodSpec{
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
						}),
					)),
			)),
	}, {
		name: "Valid EventListener with custom resources",
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
					TriggerRef: "triggerref",
				}},
				Resources: v1alpha1.Resources{
					CustomResource: &v1alpha1.CustomResource{
						RawExtension: getValidRawData(t),
					},
				},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.el.Validate(context.Background())
			if err != nil {
				t.Errorf("EventListener.Validate() expected no error, but got one, EventListener: %v, error: %v", test.el, err)
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
		el: bldr.EventListener("longlonglonglonglonglonglonglonglonglonglonglonglonglonglname", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1"))),
	}, {
		name: "Valid EventListener with empty TriggerTemplate name",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("", "v1alpha1"))),
	}, {
		name: "TriggerBinding with no ref or spec",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("", "", "v1alpha1"),
				))),
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
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerTriggerInterceptor("svc", "", "", ""),
				))),
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
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "NamespaceTriggerBinding", "v1alpha1"),
				))),
	}, {
		name: "Interceptor Wrong APIVersion",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerTriggerInterceptor("foo", "v3", "Service", ""),
				))),
	}, {
		name: "Interceptor Wrong Kind",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerTriggerInterceptor("foo", "v1", "Deployment", ""),
				))),
	}, {
		name: "Interceptor Non-Canonical Header",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerTriggerInterceptor("foo", "v1", "Deployment", "",
						bldr.EventInterceptorParam("non-canonical-header-key", "valid value"),
					),
				))),
	}, {
		name: "Interceptor Empty Header Name",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerTriggerInterceptor("foo", "v1", "Deployment", "",
						bldr.EventInterceptorParam("", "valid value"),
					),
				))),
	}, {
		name: "Interceptor Empty Header Value",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerTriggerInterceptor("foo", "v1", "Deployment", "",
						bldr.EventInterceptorParam("Valid-Header-Key", ""),
					),
				))),
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
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerCELInterceptor("body.value == 'test')"),
				))),
	}, {
		name: "CEL interceptor with bad overlay expression",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerCELInterceptor("", bldr.EventListenerCELOverlay("body.value", "'testing')")),
				))),
	}, {
		name: "Triggers name has invalid label characters",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerTriggerName("github.com/tektoncd/triggers"),
				))),
	}, {
		name: "Triggers name is longer than the allowable label value (63 characters)",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "", "v1alpha1"),
					bldr.EventListenerTriggerName("1234567890123456789012345678901234567890123456789012345678901234"),
				))),
	}, {
		name: "user specify invalid replicas",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerReplicas(-1),
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "TriggerBinding", "v1alpha1"),
				))),
	}, {
		name: "user specify multiple containers",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "TriggerBinding", "v1alpha1"),
				),
				bldr.EventListenerResources(
					bldr.EventListenerKubernetesResources(
						bldr.EventListenerPodSpec(duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Env: []corev1.EnvVar{{Name: "HTTP"}},
									}, {
										Env: []corev1.EnvVar{{Name: "TCP"}},
									}},
								},
							},
						}),
					)),
			)),
	}, {
		name: "user specifies an unsupported podspec field",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "TriggerBinding", "v1alpha1"),
				),
				bldr.EventListenerResources(
					bldr.EventListenerKubernetesResources(
						bldr.EventListenerPodSpec(duckv1.WithPodSpec{
							Template: duckv1.PodSpecable{
								Spec: corev1.PodSpec{
									NodeName: "minikube",
								},
							},
						}),
					)),
			)),
	}, {
		name: "user specifies an unsupported container fields",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "TriggerBinding", "v1alpha1"),
				),
				bldr.EventListenerResources(
					bldr.EventListenerKubernetesResources(
						bldr.EventListenerPodSpec(duckv1.WithPodSpec{
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
						}),
					)),
			)),
	}, {
		name: "user specifies an invalid env for TLS connection",
		el: bldr.EventListener("name", "namespace",
			bldr.EventListenerSpec(
				bldr.EventListenerTrigger("tt", "v1alpha1",
					bldr.EventListenerTriggerBinding("tb", "TriggerBinding", "v1alpha1"),
				),
				bldr.EventListenerResources(
					bldr.EventListenerKubernetesResources(
						bldr.EventListenerPodSpec(duckv1.WithPodSpec{
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
						}),
					)),
			)),
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
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.el.Validate(context.Background()); err == nil {
				t.Errorf("EventListener.Validate() expected error, but get none, EventListener: %v", test.el)
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
