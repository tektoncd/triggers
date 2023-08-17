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
	"strings"
	"testing"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/ptr"
)

func Test_TriggerValidate_OnDelete(t *testing.T) {
	tr := &v1alpha1.Trigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.Repeat("foo", 64), // Length should be lower than 63
			Namespace: "namespace",
		},
		Spec: v1alpha1.TriggerSpec{
			// Binding with no spec is invalid, but shouldn't block the delete
			Bindings: []*v1alpha1.TriggerSpecBinding{{Name: "", Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "", APIVersion: "v1alpha1"}},
			Template: v1alpha1.TriggerSpecTemplate{Ref: ptr.String("tt")},
		},
	}
	err := tr.Validate(apis.WithinDelete(context.Background()))
	if err != nil {
		t.Errorf("Trigger.Validate() on Delete expected no error, but got one, Trigger: %v, error: %v", tr, err)
	}
}

func Test_TriggerValidate(t *testing.T) {
	tests := []struct {
		name string
		tr   *v1alpha1.Trigger
	}{{
		name: "Valid Trigger No TriggerBinding",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Template: v1alpha1.TriggerSpecTemplate{
					Ref:        ptr.String("tt"),
					APIVersion: "v1alpha1",
				},
			},
		},
	}, {
		name: "Valid Trigger with TriggerBinding",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Ref:        "tb",
					Kind:       v1alpha1.NamespacedTriggerBindingKind,
					APIVersion: "v1alpha1",
				}},
				Template: v1alpha1.TriggerSpecTemplate{
					Ref:        ptr.String("tt"),
					APIVersion: "v1alpha1",
				},
			},
		},
	}, {
		name: "Valid Trigger with ClusterTriggerBinding",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Ref:        "tb",
					Kind:       v1alpha1.ClusterTriggerBindingKind,
					APIVersion: "v1alpha1",
				}},
				Template: v1alpha1.TriggerSpecTemplate{
					Ref:        ptr.String("tt"),
					APIVersion: "v1alpha1",
				},
			},
		},
	}, {
		name: "Valid Trigger with multiple TriggerBindings",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Ref:        "tb",
					Kind:       v1alpha1.NamespacedTriggerBindingKind,
					APIVersion: "v1alpha1",
				}, {
					Ref:        "tb",
					Kind:       v1alpha1.ClusterTriggerBindingKind,
					APIVersion: "v1alpha1",
				}, {
					Ref:        "tb2",
					Kind:       v1alpha1.NamespacedTriggerBindingKind,
					APIVersion: "v1alpha1",
				}},
				Template: v1alpha1.TriggerSpecTemplate{
					Ref:        ptr.String("tt"),
					APIVersion: "v1alpha1",
				},
			},
		},
	}, {
		name: "Trigger with new embedded TriggerBindings",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: "ns",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Name:  "param1",
					Value: ptr.String("val1"),
				}, {
					Name:  "param2",
					Value: ptr.String("val2"),
				}, {
					Ref:  "ref-to-another-binding",
					Kind: v1alpha1.NamespacedTriggerBindingKind,
				}},
				Template: v1alpha1.TriggerSpecTemplate{Ref: ptr.String("baz")},
			},
		},
	}, {
		name: "Valid Trigger Interceptor",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Interceptors: []*v1alpha1.TriggerInterceptor{{
					Ref: v1alpha1.InterceptorRef{
						Name:       "cel",
						Kind:       v1alpha1.ClusterInterceptorKind,
						APIVersion: "v1alpha1",
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
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Ref:        "tb",
					Kind:       v1alpha1.ClusterTriggerBindingKind,
					APIVersion: "v1alpha1",
				}},
				Template: v1alpha1.TriggerSpecTemplate{
					Ref:        ptr.String("tt"),
					APIVersion: "v1alpha1",
				},
			},
		},
	}, {
		name: "Valid Trigger with no trigger name",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
				Name:      "name",
			},
			Spec: v1alpha1.TriggerSpec{
				Template: v1alpha1.TriggerSpecTemplate{
					Ref:        ptr.String("tt"),
					APIVersion: "v1alpha1",
				},
			},
		},
	}, {
		name: "Trigger with embedded Template",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name: "name",
			},
			Spec: v1alpha1.TriggerSpec{
				Template: v1alpha1.TriggerSpecTemplate{
					Spec: &v1alpha1.TriggerTemplateSpec{
						Params: []v1alpha1.ParamSpec{{
							Name: "tparam",
						}},
						ResourceTemplates: []v1alpha1.TriggerResourceTemplate{{
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
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-trigger",
			},
			Spec: v1alpha1.TriggerSpec{
				Template: v1alpha1.TriggerSpecTemplate{
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
		tr   *v1alpha1.Trigger
	}{{
		name: "TriggerBinding with no spec",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
		},
	}, {
		name: "Bindings missing ref",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{Name: "", Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "", APIVersion: "v1alpha1"}},
				Template: v1alpha1.TriggerSpecTemplate{Ref: ptr.String("tt")},
			},
		},
	}, {
		name: "Bindings with ref missing kind",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{Name: "tb", Kind: "", Ref: "tb", APIVersion: "v1alpha1"}},
				Template: v1alpha1.TriggerSpecTemplate{Ref: ptr.String("tt")},
			},
		},
	}, {
		name: "Bindings with ref wrong kind",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{Kind: "BadKind", Ref: "tb", APIVersion: "v1alpha1"}},
				Template: v1alpha1.TriggerSpecTemplate{Ref: ptr.String("tt")},
			},
		},
	}, {
		name: "Bindings with name but no value",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "ns",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{Name: "foo"}},
				Template: v1alpha1.TriggerSpecTemplate{Ref: ptr.String("tt")},
			},
		},
	}, {
		name: "Template with wrong apiVersion",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{Name: "tb", Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb", APIVersion: "v1alpha1"}},
				Template: v1alpha1.TriggerSpecTemplate{Ref: ptr.String("tt"), APIVersion: "invalid"},
			},
		},
	}, {
		name: "Template with nil Ref",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{Name: "tb", Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb", APIVersion: "v1alpha1"}},
				Template: v1alpha1.TriggerSpecTemplate{Ref: nil, APIVersion: "v1alpha1"},
			},
		},
	}, {
		name: "Template with empty Ref",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{Name: "tb", Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb", APIVersion: "v1alpha1"}},
				Template: v1alpha1.TriggerSpecTemplate{Ref: ptr.String(""), APIVersion: "v1alpha1"},
			},
		},
	}, {
		name: "Valid Trigger with invalid TriggerBinding",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Ref:        "tb",
					Kind:       "BADBINDINGKIND",
					APIVersion: "v1alpha1",
				}},
				Template: v1alpha1.TriggerSpecTemplate{
					Ref:        ptr.String("tt"),
					APIVersion: "v1alpha1",
				},
			},
		},
	}, {
		name: "Trigger template with both ref and spec",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{Name: "name"},
			Spec: v1alpha1.TriggerSpec{
				Template: v1alpha1.TriggerSpecTemplate{
					Ref: ptr.String("ttname"),
					Spec: &v1alpha1.TriggerTemplateSpec{
						ResourceTemplates: []v1alpha1.TriggerResourceTemplate{{
							RawExtension: simpleResourceTemplate(t),
						}},
					},
				},
			},
		},
	}, {
		name: "Trigger template with both name and spec",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{Name: "name"},
			Spec: v1alpha1.TriggerSpec{
				Template: v1alpha1.TriggerSpecTemplate{
					Ref: ptr.String("tt-name"),
					Spec: &v1alpha1.TriggerTemplateSpec{
						ResourceTemplates: []v1alpha1.TriggerResourceTemplate{{
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
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{Name: "name"},
			Spec: v1alpha1.TriggerSpec{
				Template: v1alpha1.TriggerSpecTemplate{},
			},
		},
	}, {
		name: "Trigger template with invalid spec",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{Name: "name"},
			Spec: v1alpha1.TriggerSpec{
				Template: v1alpha1.TriggerSpecTemplate{
					Spec: &v1alpha1.TriggerTemplateSpec{
						ResourceTemplates: []v1alpha1.TriggerResourceTemplate{{}},
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
