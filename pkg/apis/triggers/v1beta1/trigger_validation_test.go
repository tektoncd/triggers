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

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/ptr"
)

func Test_TriggerValidate(t *testing.T) {
	tests := []struct {
		name string
		tr   *v1beta1.Trigger
	}{{
		name: "Valid Trigger No TriggerBinding",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerSpec{
				Template: v1beta1.TriggerSpecTemplate{
					Ref:        ptr.String("tt"),
					APIVersion: "v1beta1",
				},
			},
		},
	}, {
		name: "Valid Trigger with TriggerBinding",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{
					Ref:        "tb",
					Kind:       v1beta1.NamespacedTriggerBindingKind,
					APIVersion: "v1beta1",
				}},
				Template: v1beta1.TriggerSpecTemplate{
					Ref:        ptr.String("tt"),
					APIVersion: "v1beta1",
				},
			},
		},
	}, {
		name: "Valid Trigger with ClusterTriggerBinding",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{
					Ref:        "tb",
					Kind:       v1beta1.ClusterTriggerBindingKind,
					APIVersion: "v1beta1",
				}},
				Template: v1beta1.TriggerSpecTemplate{
					Ref:        ptr.String("tt"),
					APIVersion: "v1beta1",
				},
			},
		},
	}, {
		name: "Valid Trigger with multiple TriggerBindings",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{
					Ref:        "tb",
					Kind:       v1beta1.NamespacedTriggerBindingKind,
					APIVersion: "v1beta1",
				}, {
					Ref:        "tb",
					Kind:       v1beta1.ClusterTriggerBindingKind,
					APIVersion: "v1beta1",
				}, {
					Ref:        "tb2",
					Kind:       v1beta1.NamespacedTriggerBindingKind,
					APIVersion: "v1beta1",
				}},
				Template: v1beta1.TriggerSpecTemplate{
					Ref:        ptr.String("tt"),
					APIVersion: "v1beta1",
				},
			},
		},
	}, {
		name: "Trigger with new embedded TriggerBindings",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: "ns",
			},
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{
					Name:  "param1",
					Value: ptr.String("val1"),
				}, {
					Name:  "param2",
					Value: ptr.String("val2"),
				}, {
					Ref:  "ref-to-another-binding",
					Kind: v1beta1.NamespacedTriggerBindingKind,
				}},
				Template: v1beta1.TriggerSpecTemplate{Ref: ptr.String("baz")},
			},
		},
	}, {
		name: "Valid Trigger Interceptor",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerSpec{
				Interceptors: []*v1beta1.TriggerInterceptor{{
					Ref: v1beta1.InterceptorRef{
						Name:       "cel",
						Kind:       v1beta1.ClusterInterceptorKind,
						APIVersion: "v1beta1",
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
				Bindings: []*v1beta1.TriggerSpecBinding{{
					Ref:        "tb",
					Kind:       v1beta1.ClusterTriggerBindingKind,
					APIVersion: "v1beta1",
				}},
				Template: v1beta1.TriggerSpecTemplate{
					Ref:        ptr.String("tt"),
					APIVersion: "v1beta1",
				},
			},
		},
	}, {
		name: "Valid Trigger with no trigger name",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
				Name:      "name",
			},
			Spec: v1beta1.TriggerSpec{
				Template: v1beta1.TriggerSpecTemplate{
					Ref:        ptr.String("tt"),
					APIVersion: "v1beta1",
				},
			},
		},
	}, {
		name: "Trigger with embedded Template",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name: "name",
			},
			Spec: v1beta1.TriggerSpec{
				Template: v1beta1.TriggerSpecTemplate{
					Spec: &v1beta1.TriggerTemplateSpec{
						Params: []v1beta1.ParamSpec{{
							Name: "tparam",
						}},
						ResourceTemplates: []v1beta1.TriggerResourceTemplate{{
							RawExtension: test.RawExtension(t, pipelinev1.PipelineRun{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "tekton.dev/v1beta1",
									Kind:       "PipelineRun",
								},
							}),
						}},
					},
				},
			},
		},
	}, {
		name: "Trigger referenced with deprecated name field", // TODO(#FIXME): Remove when Name is removed.
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-trigger",
			},
			Spec: v1beta1.TriggerSpec{
				Template: v1beta1.TriggerSpecTemplate{
					Ref: ptr.String("ref-to-a-template"),
				},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.tr.Validate(context.Background())
			if err != nil {
				t.Errorf("Trigger.Validate() expected no error, but got one, Trigger: %v, error: %v", test.tr, err)
			}
		})
	}
}

func TestTriggerValidate_error(t *testing.T) {
	tests := []struct {
		name string
		tr   *v1beta1.Trigger
	}{{
		name: "TriggerBinding with no spec",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
		},
	}, {
		name: "Bindings missing ref",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{Name: "", Kind: v1beta1.NamespacedTriggerBindingKind, Ref: "", APIVersion: "v1beta1"}},
				Template: v1beta1.TriggerSpecTemplate{Ref: ptr.String("tt")},
			},
		},
	}, {
		name: "Bindings with ref missing kind",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{Name: "tb", Kind: "", Ref: "tb", APIVersion: "v1beta1"}},
				Template: v1beta1.TriggerSpecTemplate{Ref: ptr.String("tt")},
			},
		},
	}, {
		name: "Bindings with ref wrong kind",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{Kind: "BadKind", Ref: "tb", APIVersion: "v1beta1"}},
				Template: v1beta1.TriggerSpecTemplate{Ref: ptr.String("tt")},
			},
		},
	}, {
		name: "Bindings with name but no value",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "ns",
			},
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{Name: "foo"}},
				Template: v1beta1.TriggerSpecTemplate{Ref: ptr.String("tt")},
			},
		},
	}, {
		name: "Template with wrong apiVersion",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{Name: "tb", Kind: v1beta1.NamespacedTriggerBindingKind, Ref: "tb", APIVersion: "v1beta1"}},
				Template: v1beta1.TriggerSpecTemplate{Ref: ptr.String("tt"), APIVersion: "invalid"},
			},
		},
	}, {
		name: "Template with nil Ref",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{Name: "tb", Kind: v1beta1.NamespacedTriggerBindingKind, Ref: "tb", APIVersion: "v1beta1"}},
				Template: v1beta1.TriggerSpecTemplate{Ref: nil, APIVersion: "v1beta1"},
			},
		},
	}, {
		name: "Template with empty Ref",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{Name: "tb", Kind: v1beta1.NamespacedTriggerBindingKind, Ref: "tb", APIVersion: "v1beta1"}},
				Template: v1beta1.TriggerSpecTemplate{Ref: ptr.String(""), APIVersion: "v1beta1"},
			},
		},
	}, {
		name: "Valid Trigger with invalid TriggerBinding",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{
					Ref:        "tb",
					Kind:       "BADBINDINGKIND",
					APIVersion: "v1beta1",
				}},
				Template: v1beta1.TriggerSpecTemplate{
					Ref:        ptr.String("tt"),
					APIVersion: "v1beta1",
				},
			},
		},
	}, {
		name: "Trigger template with both ref and spec",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{Name: "name"},
			Spec: v1beta1.TriggerSpec{
				Template: v1beta1.TriggerSpecTemplate{
					Ref: ptr.String("ttname"),
					Spec: &v1beta1.TriggerTemplateSpec{
						ResourceTemplates: []v1beta1.TriggerResourceTemplate{{
							RawExtension: simpleResourceTemplate(t),
						}},
					},
				},
			},
		},
	}, {
		name: "Trigger template with both name and spec",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{Name: "name"},
			Spec: v1beta1.TriggerSpec{
				Template: v1beta1.TriggerSpecTemplate{
					Ref: ptr.String("tt-name"),
					Spec: &v1beta1.TriggerTemplateSpec{
						ResourceTemplates: []v1beta1.TriggerResourceTemplate{{
							RawExtension: test.RawExtension(t, pipelinev1.PipelineRun{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "tekton.dev/v1beta1",
									Kind:       "PipelineRun",
								},
							}),
						}},
					},
				},
			},
		},
	}, {
		name: "Trigger template missing both ref and spec",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{Name: "name"},
			Spec: v1beta1.TriggerSpec{
				Template: v1beta1.TriggerSpecTemplate{},
			},
		},
	}, {
		name: "Trigger template with invalid spec",
		tr: &v1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{Name: "name"},
			Spec: v1beta1.TriggerSpec{
				Template: v1beta1.TriggerSpecTemplate{
					Spec: &v1beta1.TriggerTemplateSpec{
						ResourceTemplates: []v1beta1.TriggerResourceTemplate{{}},
					},
				},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.tr.Validate(context.Background()); err == nil {
				t.Errorf("Trigger.Validate() expected error, but get none, Trigger: %v", test.tr)
			}
		})
	}
}
