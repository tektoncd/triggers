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

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	bldr "github.com/tektoncd/triggers/test/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				bldr.TriggerSpecBinding("tb", "TriggerBinding", "tb", "v1alpha1"),
			)),
	}, {
		name: "Valid Trigger with ClusterTriggerBinding",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "ClusterTriggerBinding", "tb", "v1alpha1"),
			)),
	}, {
		name: "Valid Trigger with multiple TriggerBindings",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "ClusterTriggerBinding", "tb", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "TriggerBinding", "tb", "v1alpha1"),
				bldr.TriggerSpecBinding("tb3", "", "tb3", "v1alpha1"),
			)),
	}, {
		name: "Valid Trigger Interceptor",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "tb", "v1alpha1"),
				bldr.TriggerSpecInterceptor("svc", "v1", "Service", "namespace"),
			)),
	}, {
		name: "Valid Trigger Interceptor With Header",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "tb", "v1alpha1"),
				bldr.TriggerSpecInterceptor("svc", "v1", "Service", "namespace",
					bldr.TriggerSpecInterceptorParam("Valid-Header-Key", "valid value"),
				),
			)),
	}, {
		name: "Valid Trigger Interceptor With Headers",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "tb", "v1alpha1"),
				bldr.TriggerSpecInterceptor("svc", "v1", "Service", "namespace",
					bldr.TriggerSpecInterceptorParam("Valid-Header-Key1", "valid value1"),
					bldr.TriggerSpecInterceptorParam("Valid-Header-Key1", "valid value2"),
					bldr.TriggerSpecInterceptorParam("Valid-Header-Key2", "valid value"),
				),
			)),
	}, {
		name: "Valid Trigger with CTR interceptor",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "tb", "v1alpha1"),
				bldr.TriggerSpecCELInterceptor("body.value == 'test'"),
			)),
	}, {
		name: "Valid Trigger with no trigger name",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "tb", "v1alpha1"),
			)),
	}, {
		name: "Valid Trigger with embedded bindings",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("", "", "", "v1alpha1", bldr.TriggerBindingParam("key", "value")),
			)),
	}, {
		name: "Valid Trigger with CEL overlays",
		tr: bldr.Trigger("name", "namespace",
			bldr.TriggerSpec(
				bldr.TriggerSpecTemplate("tt", "v1alpha1"),
				bldr.TriggerSpecBinding("tb", "", "tb", "v1alpha1"),
				bldr.TriggerSpecCELInterceptor("", bldr.TriggerSpecCELOverlay("body.value", "'testing'")),
			)),
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
				Template: v1alpha1.TriggerSpecTemplate{Name: "tt"},
			},
		},
	}, {
		name: "Bindings missing kind",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{Name: "tb", Kind: "", Ref: "tb", APIVersion: "v1alpha1"}},
				Template: v1alpha1.TriggerSpecTemplate{Name: "tt"},
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
				Template: v1alpha1.TriggerSpecTemplate{Name: "tt", APIVersion: "invalid"},
			},
		},
	}, {
		name: "Template with missing name",
		tr: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				Bindings: []*v1alpha1.TriggerSpecBinding{{Name: "tb", Kind: v1alpha1.NamespacedTriggerBindingKind, Ref: "tb", APIVersion: "v1alpha1"}},
				Template: v1alpha1.TriggerSpecTemplate{Name: "", APIVersion: "v1alpha1"},
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
				Template:     v1alpha1.TriggerSpecTemplate{Name: "tt", APIVersion: "v1alpha1"},
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
				Template: v1alpha1.TriggerSpecTemplate{Name: "tt", APIVersion: "v1alpha1"},
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
				Template: v1alpha1.TriggerSpecTemplate{Name: "tt"},
				Interceptors: []*v1alpha1.TriggerInterceptor{{
					GitHub:    &v1alpha1.GitHubInterceptor{},
					GitLab:    &v1alpha1.GitLabInterceptor{},
					Bitbucket: &v1alpha1.BitbucketInterceptor{},
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
				Template: v1alpha1.TriggerSpecTemplate{Name: "tt"},
				Interceptors: []*v1alpha1.TriggerInterceptor{{
					CEL: &v1alpha1.CELInterceptor{},
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
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.tr.Validate(context.Background()); err == nil {
				t.Errorf("Trigger.Validate() expected error, but get none, Trigger: %v", test.tr)
			}
		})
	}
}
