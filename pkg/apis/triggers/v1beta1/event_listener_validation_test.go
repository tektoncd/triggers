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
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/pkg/apis/triggers"
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

var myObjectMeta = metav1.ObjectMeta{
	Name:      "name",
	Namespace: "namespace",
}

func Test_EventListenerValidate(t *testing.T) {
	ctxWithAlphaFieldsEnabled, err := test.FeatureFlagsToContext(context.Background(), map[string]string{
		"enable-api-fields": "alpha",
	})
	if err != nil {
		t.Fatalf("unexpected error initializing feature flags: %v", err)
	}

	tests := []struct {
		name    string
		el      *triggersv1beta1.EventListener
		ctx     context.Context
		wantErr *apis.FieldError
	}{{
		name: "TriggerTemplate Does Not Exist",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: myObjectMeta,
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
			ObjectMeta: myObjectMeta,
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
			ObjectMeta: myObjectMeta,
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					TriggerRef: "tt",
				}},
			},
		},
	}, {
		name: "Valid EventListener with Annotation",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "name",
				Namespace:   "namespace",
				Annotations: map[string]string{triggers.PayloadValidationAnnotation: "true"},
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
			ObjectMeta: myObjectMeta,
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
			ObjectMeta: myObjectMeta,
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
			ObjectMeta: myObjectMeta,
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
			ObjectMeta: myObjectMeta,
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
			ObjectMeta: myObjectMeta,
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Webhook: &triggersv1beta1.WebhookInterceptor{
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
			ObjectMeta: myObjectMeta,
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
			ObjectMeta: myObjectMeta,
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
			ObjectMeta: myObjectMeta,
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
			ObjectMeta: myObjectMeta,
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
			ObjectMeta: myObjectMeta,
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
			ObjectMeta: myObjectMeta,
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("tt"),
					},
				}},
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
			ObjectMeta: myObjectMeta,
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
			ObjectMeta: myObjectMeta,
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
		ctx:  ctxWithAlphaFieldsEnabled,
		el: &triggersv1beta1.EventListener{
			ObjectMeta: myObjectMeta,
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
								RawExtension: test.RawExtension(t, pipelinev1beta1.PipelineRun{
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
	}, {
		name: "Valid event listener with TriggerGroup and namespaceSelector",
		ctx:  ctxWithAlphaFieldsEnabled,
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				TriggerGroups: []triggersv1beta1.EventListenerTriggerGroup{{
					Name: "my-group",
					Interceptors: []*triggersv1beta1.TriggerInterceptor{{
						Ref: triggersv1beta1.InterceptorRef{
							Name: "cel",
						},
						Params: []triggersv1beta1.InterceptorParams{{
							Name:  "filter",
							Value: test.ToV1JSON(t, "has(body.repository)"),
						}},
					}},
					TriggerSelector: triggersv1beta1.EventListenerTriggerSelector{
						NamespaceSelector: triggersv1beta1.NamespaceSelector{
							MatchNames: []string{
								"foobar",
							},
						},
					},
				}},
			},
		}}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.ctx
			if ctx == nil {
				ctx = context.Background()
			}
			err := tc.el.Validate(ctx)
			if err != nil {
				t.Errorf("EventListener.Validate() expected no error, but got one, EventListener: %v, error: %v", tc.el, err)
			}
		})
	}
}

func TestEventListenerValidate_error(t *testing.T) {
	ctxWithAlphaFieldsEnabled, err := test.FeatureFlagsToContext(context.Background(), map[string]string{
		"enable-api-fields": "alpha",
	})
	if err != nil {
		t.Fatalf("unexpected error initializing feature flags: %v", err)
	}

	tests := []struct {
		name    string
		el      *triggersv1beta1.EventListener
		ctx     context.Context
		wantErr *apis.FieldError
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
		wantErr: apis.ErrInvalidValue(`eventListener name 'longlonglonglonglonglonglonglonglonglonglonglonglonglonglname' must be no more than 60 characters long`, "metadata.name"),
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
		wantErr: apis.ErrMissingField("spec.triggers[0].template.ref"),
	}, {
		name: "Invalid Annotation value",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "name",
				Namespace:   "namespace",
				Annotations: map[string]string{triggers.PayloadValidationAnnotation: "xyz"},
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					TriggerRef: "tt",
				}},
			},
		},
		wantErr: apis.ErrInvalidValue(`tekton.dev/payload-validation annotation must have value 'true' or 'false'`, "metadata.annotations"),
	}, {
		name: "TriggerBinding with no ref or spec or name",
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
		wantErr: apis.ErrMissingOneOf("spec.triggers[0].bindings[0].name, spec.triggers[0].bindings[0].ref, spec.triggers[0].bindings[0].spec"),
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
		wantErr: apis.ErrMissingOneOf("spec.triggers[0].bindings[0].name, spec.triggers[0].bindings[0].ref, spec.triggers[0].bindings[0].spec"),
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
		wantErr: apis.ErrInvalidValue("invalid kind", "spec.triggers[0].bindings[0].kind"),
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
		wantErr: apis.ErrInvalidValue("invalid apiVersion", "spec.triggers[0].template.apiVersion"),
	}, {
		name: "Template with both ref and name",
		el: &triggersv1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Bindings: []*triggersv1beta1.EventListenerBinding{{Kind: triggersv1beta1.NamespacedTriggerBindingKind, Ref: "tb", Name: "somename"}},
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt"), APIVersion: "v1beta1"},
				}},
			},
		},
		wantErr: apis.ErrMultipleOneOf("spec.triggers[0].bindings[0].name", "spec.triggers[0].bindings[0].ref"),
	}, {
		name: "Interceptor Name and APIVersion only",
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
							ObjectRef: &corev1.ObjectReference{Name: "svc", APIVersion: "v1"},
						},
					}},
				}},
			},
		},
		wantErr: apis.ErrMissingField("spec.triggers[0].interceptors[0].interceptor.webhook.objectRef.kind"),
	}, {
		name: "Interceptor Name and Kind only",
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
							ObjectRef: &corev1.ObjectReference{Name: "svc", Kind: "Service"},
						},
					}},
				}},
			},
		},
		wantErr: apis.ErrMissingField("spec.triggers[0].interceptors[0].interceptor.webhook.objectRef.apiVersion"),
	}, {
		name: "Interceptor Missing Interceptor",
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
		wantErr: apis.ErrMissingField("spec.triggers[0].interceptors[0].interceptor"),
	}, {
		name: "Interceptor Missing ObjectRef",
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
						Webhook: &triggersv1beta1.WebhookInterceptor{},
					}},
				}},
			},
		},
		wantErr: apis.ErrMissingField("spec.triggers[0].interceptors[0].interceptor.webhook.objectRef"),
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
		wantErr: func() *apis.FieldError {
			var errs *apis.FieldError
			errs = errs.Also(apis.ErrMissingField("spec.triggers[0].interceptors[0].interceptor.webhook.objectRef.apiVersion"))
			errs = errs.Also(apis.ErrMissingField("spec.triggers[0].interceptors[0].interceptor.webhook.objectRef.kind"))
			return errs
		}(),
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
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt"), APIVersion: "v1beta1"},
				}},
			},
		},
		wantErr: apis.ErrInvalidValue("invalid kind", "spec.triggers[0].bindings[0].kind"),
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
		wantErr: apis.ErrInvalidValue("invalid apiVersion", "spec.triggers[0].interceptors[0].interceptor.webhook.objectRef.apiVersion"),
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
		wantErr: apis.ErrInvalidValue("invalid kind", "spec.triggers[0].interceptors[0].interceptor.webhook.objectRef.kind"),
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
		wantErr: apis.ErrInvalidValue("invalid header name", "spec.triggers[0].interceptors[0].interceptor.webhook.header[0].name"),
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
		wantErr: apis.ErrInvalidValue("invalid header name", "spec.triggers[0].interceptors[0].interceptor.webhook.header[0].name"),
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
		wantErr: apis.ErrInvalidValue("invalid header value", "spec.triggers[0].interceptors[0].interceptor.webhook.header[0].value"),
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
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt"), APIVersion: "v1beta1"},
				}},
			},
		},
		wantErr: apis.ErrInvalidValue(`trigger name 'github.com/tektoncd/triggers' must be a valid label value`, "spec.triggers[0].name"),
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
					Template: &triggersv1beta1.EventListenerTemplate{Ref: ptr.String("tt"), APIVersion: "v1beta1"},
				}},
			},
		},
		wantErr: apis.ErrInvalidValue(`trigger name '1234567890123456789012345678901234567890123456789012345678901234' must be a valid label value`, "spec.triggers[0].name"),
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
		wantErr: apis.ErrInvalidValue(-1, "spec.resources.kubernetesResource.spec.replicas"),
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
		wantErr: apis.ErrMultipleOneOf("spec.resources.kubernetesResource.spec.template.spec.containers"),
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
		wantErr: apis.ErrDisallowedFields("spec.resources.kubernetesResource.spec.template.spec.nodeName"),
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
		wantErr: func() *apis.FieldError {
			var errs *apis.FieldError
			errs = errs.Also(apis.ErrDisallowedFields("spec.resources.kubernetesResource.spec.template.spec.containers[0].env[0].value"))
			errs = errs.Also(apis.ErrDisallowedFields("spec.resources.kubernetesResource.spec.template.spec.containers[0].env[1].valueFrom.configMapKeyRef"))
			errs = errs.Also(apis.ErrDisallowedFields("spec.resources.kubernetesResource.spec.template.spec.containers[0].name"))
			return errs
		}(),
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
		wantErr: apis.ErrGeneric("Expected env's are TLS_CERT and TLS_KEY, but got only one env TLS_CERT"),
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
		wantErr: apis.ErrMultipleOneOf("spec.resources.customResource", "spec.resources.kubernetesResource"),
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
						APIVersion: "v1beta1",
					}},
				}},
				Resources: triggersv1beta1.Resources{
					CustomResource: &triggersv1beta1.CustomResource{
						RawExtension: getRawData(t),
					},
				},
			},
		},
		wantErr: func() *apis.FieldError {
			var errs *apis.FieldError
			errs = errs.Also(apis.ErrMultipleOneOf("spec.resources.customResource.spec.template.spec.containers"))
			errs = errs.Also(apis.ErrMissingOneOf("spec.triggers[0].template", "spec.triggers[0].triggerRef"))
			errs = errs.Also(apis.ErrDisallowedFields("spec.resources.customResource.spec.template.spec.nodeName"))
			return errs
		}(),
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
		wantErr: apis.ErrMultipleOneOf("spec.triggers[0].template or bindings or interceptors", "spec.triggers[0].triggerRef"),
	},
		{
			name: "missing label and namespace selector",
			ctx:  ctxWithAlphaFieldsEnabled,
			el: &triggersv1beta1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: triggersv1beta1.EventListenerSpec{
					TriggerGroups: []triggersv1beta1.EventListenerTriggerGroup{{
						Name: "my-group",
						Interceptors: []*triggersv1beta1.TriggerInterceptor{{
							Ref: triggersv1beta1.InterceptorRef{
								Name: "cel",
							},
							Params: []triggersv1beta1.InterceptorParams{{
								Name:  "filter",
								Value: test.ToV1JSON(t, "has(body.repository)"),
							}},
						}},
					}},
				},
			},
			wantErr: apis.ErrMissingOneOf("spec.triggerGroups[0].triggerSelector.labelSelector", "spec.triggerGroups[0].triggerSelector.namespaceSelector"),
		}, {
			name: "triggerGroup requires interceptor",
			ctx:  ctxWithAlphaFieldsEnabled,
			el: &triggersv1beta1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: triggersv1beta1.EventListenerSpec{
					TriggerGroups: []triggersv1beta1.EventListenerTriggerGroup{{
						Name: "my-group",
						TriggerSelector: triggersv1beta1.EventListenerTriggerSelector{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"foo": "bar",
								},
							},
						},
					}},
				},
			},
			wantErr: apis.ErrMissingField("spec.triggerGroups[0].interceptors"),
		}, {
			name: "empty spec for eventlistener",
			ctx:  ctxWithAlphaFieldsEnabled,
			el: &triggersv1beta1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
			},
			wantErr: apis.ErrMissingOneOf("spec.labelSelector", "spec.namespaceSelector", "spec.triggerGroups", "spec.triggers"),
		}, {
			name: "invalid interceptor for eventlistener",
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
						Name: "test",
						Interceptors: []*triggersv1beta1.EventInterceptor{{
							Ref: triggersv1beta1.InterceptorRef{
								Name: "cel",
							},
						},
							nil,
						},
					}},
				},
			},
			wantErr: &apis.FieldError{
				Message: "invalid value: interceptor '<nil>' must be a valid value",
				Paths:   []string{"spec.triggers[0].interceptors[1]"},
			},
		}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.ctx
			if ctx == nil {
				ctx = context.Background()
			}
			got := tc.el.Validate(ctx)

			if diff := cmp.Diff(tc.wantErr.Error(), got.Error()); diff != "" {
				t.Error("EventListener.Validate() (-want, +got) =", diff)
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
