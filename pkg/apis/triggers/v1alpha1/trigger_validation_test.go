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
	"testing"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/test"
	bldr "github.com/tektoncd/triggers/test/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/pkg/ptr"
)

func Test_TriggerValidate(t *testing.T) {
	tests := []struct {
		name string
		tr   *v1alpha1.Trigger
	}{{
		name: "Valid Trigger No TriggerBinding",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"))),
	}, {
		name: "Valid Trigger with TriggerBinding",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "TriggerBinding", "", "v1alpha1"),
			)),
	}, {
		name: "Valid Trigger with ClusterTriggerBinding",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "ClusterTriggerBinding", "", "v1alpha1"),
			)),
	}, {
		name: "Valid Trigger with multiple TriggerBindings",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "ClusterTriggerBinding", "", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "TriggerBinding", "", "v1alpha1"),
				bldr.TriggerSpecBinding("tb3", "", "", "v1alpha1"),
			)),
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
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "", "v1alpha1"),
				bldr.TriggerSpecInterceptor("svc", "v1", "Service", "namespace"),
			)),
	}, {
		name: "Valid Trigger Interceptor With Header",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "", "v1alpha1"),
				bldr.TriggerSpecInterceptor("svc", "v1", "Service", "namespace",
					bldr.TriggerSpecInterceptorParam("Valid-Header-Key", "valid value"),
				),
			)),
	}, {
		name: "Valid Trigger Interceptor With Headers",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "", "v1alpha1"),
				bldr.TriggerSpecInterceptor("svc", "v1", "Service", "namespace",
					bldr.TriggerSpecInterceptorParam("Valid-Header-Key1", "valid value1"),
					bldr.TriggerSpecInterceptorParam("Valid-Header-Key1", "valid value2"),
					bldr.TriggerSpecInterceptorParam("Valid-Header-Key2", "valid value"),
				),
			)),
	}, {
		name: "Valid Trigger with CEL interceptor",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "", "v1alpha1"),
				bldr.TriggerSpecCELInterceptor("body.value == 'test'"),
			)),
	}, {
		name: "Valid Trigger with no trigger name",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "", "v1alpha1"),
			)),
	}, {
		name: "Valid Trigger with CEL overlays",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "", "v1alpha1"),
				bldr.TriggerSpecCELInterceptor("", bldr.TriggerSpecCELOverlay("body.value", "'testing'")),
			)),
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
		tr:   bldr.Trigger("name", "namespace"),
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
		name: "Interceptor Name only",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "tb", "v1alpha1"),
				bldr.TriggerSpecInterceptor("svc", "", "", ""),
			)),
	}, {
		name: "Interceptor Missing ObjectRef",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings:     []*v1alpha1.TriggerSpecBinding{{Name: "tb", Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb", APIVersion: "v1alpha1"}},
				Template:     v1alpha1.TriggerSpecTemplate{Ref: ptr.String("tt"), APIVersion: "v1alpha1"},
				Interceptors: []*v1alpha1.TriggerInterceptor{{}},
			},
		},
	}, {
		name: "Interceptor Empty ObjectRef",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{Name: "tb", Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb", APIVersion: "v1alpha1"}},
				Template: v1alpha1.TriggerSpecTemplate{Ref: ptr.String("tt"), APIVersion: "v1alpha1"},
				Interceptors: []*v1alpha1.TriggerInterceptor{{
					Webhook: &v1alpha1.WebhookInterceptor{
						ObjectRef: &corev1.ObjectReference{
							Name: "",
						},
					},
				}},
			},
		},
	}, {
		name: "Valid Trigger with invalid TriggerBinding",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "NamespaceTriggerBinding", "tb", "v1alpha1"),
			)),
	}, {
		name: "Interceptor Wrong APIVersion",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "tb", "v1alpha1"),
				bldr.TriggerSpecInterceptor("foo", "v3", "Service", ""),
			)),
	}, {
		name: "Interceptor Wrong Kind",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "tb", "v1alpha1"),
				bldr.TriggerSpecInterceptor("foo", "v1", "Deployment", ""),
			)),
	}, {
		name: "Interceptor Non-Canonical Header",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "tb", "v1alpha1"),
				bldr.TriggerSpecInterceptor("foo", "v1", "Deployment", "",
					bldr.TriggerSpecInterceptorParam("non-canonical-header-key", "valid value"),
				),
			)),
	}, {
		name: "Interceptor Empty Header Name",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "tb", "v1alpha1"),
				bldr.TriggerSpecInterceptor("foo", "v1", "Deployment", "",
					bldr.TriggerSpecInterceptorParam("", "valid value"),
				),
			)),
	}, {
		name: "Interceptor Empty Header Value",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "tb", "v1alpha1"),
				bldr.TriggerSpecInterceptor("foo", "v1", "Deployment", "",
					bldr.TriggerSpecInterceptorParam("Valid-Header-Key", ""),
				),
			)),
	}, {
		name: "Multiple interceptors set",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{Name: "tb", Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
				Template: v1alpha1.TriggerSpecTemplate{Ref: ptr.String("tt")},
				Interceptors: []*v1alpha1.TriggerInterceptor{{
					DeprecatedGitHub:    &v1alpha1.GitHubInterceptor{},
					DeprecatedGitLab:    &v1alpha1.GitLabInterceptor{},
					DeprecatedBitbucket: &v1alpha1.BitbucketInterceptor{},
				}},
			},
		},
	}, {
		name: "CEL interceptor with no filter or overlays",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{Name: "tb", Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb"}},
				Template: v1alpha1.TriggerSpecTemplate{Ref: ptr.String("tt")},
				Interceptors: []*v1alpha1.TriggerInterceptor{{
					DeprecatedCEL: &v1alpha1.CELInterceptor{},
				}},
			},
		},
	}, {
		name: "CEL interceptor with bad filter expression",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "tb", "v1alpha1"),
				bldr.TriggerSpecCELInterceptor("body.value == 'test')"),
			)),
	}, {
		name: "CEL interceptor with bad overlay expression",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "tb", "v1alpha1"),
				bldr.TriggerSpecCELInterceptor("", bldr.TriggerSpecCELOverlay("body.value", "'testing')")),
			)),
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
		name: "Trigger template with both name and spec", // TODO(#FIXME): Remove when name field is removed
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
