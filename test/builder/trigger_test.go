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
	"knative.dev/pkg/ptr"
)

func TestTriggerBuilder(t *testing.T) {
	tests := []struct {
		name    string
		normal  *v1alpha1.Trigger
		builder *v1alpha1.Trigger
	}{{
		name:    "Empty",
		normal:  &v1alpha1.Trigger{},
		builder: Trigger("", ""),
	}, {
		name: "Name and Namespace",
		normal: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
		},
		builder: Trigger("name", "namespace"),
	}, {
		name: "One Trigger with one Binding",
		normal: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				ServiceAccountName: "serviceAccount",
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Name:       "tb1",
					Kind:       v1alpha1.NamespacedTriggerBindingKind,
					Ref:        "tb1",
					APIVersion: "v1alpha1",
				}},
				Template: v1alpha1.TriggerSpecTemplate{
					Ref:        ptr.String("tt1"),
					APIVersion: "v1alpha1",
				},
			},
		},
		builder: Trigger("name", "namespace",
			TriggerSpec(
				TriggerSpecServiceAccountName("serviceAccount"),
				TriggerSpecTemplate("tt1", "v1alpha1"),
				TriggerSpecBinding("tb1", "", "tb1", "v1alpha1"),
			),
		),
	}, {
		name: "One Trigger with ClusterTriggerBinding",
		normal: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				ServiceAccountName: "serviceAccount",
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Name:       "tb1",
					Kind:       v1alpha1.ClusterTriggerBindingKind,
					Ref:        "tb1",
					APIVersion: "v1alpha1",
				}},
				Template: v1alpha1.TriggerSpecTemplate{
					Ref:        ptr.String("tt1"),
					APIVersion: "v1alpha1",
				},
			},
		},
		builder: Trigger("name", "namespace",
			TriggerSpec(
				TriggerSpecServiceAccountName("serviceAccount"),
				TriggerSpecTemplate("tt1", "v1alpha1"),
				TriggerSpecBinding("tb1", "ClusterTriggerBinding", "tb1", "v1alpha1"),
			),
		),
	}, {
		name: "One Trigger with multiple TriggerBindings",
		normal: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				ServiceAccountName: "serviceAccount",
				Bindings: []*v1alpha1.TriggerSpecBinding{
					{
						Name: "tb1",
						Kind: v1alpha1.NamespacedTriggerBindingKind,
						Ref:  "tb1",

						APIVersion: "v1alpha1",
					},
					{
						Name:       "ctb1",
						Kind:       v1alpha1.ClusterTriggerBindingKind,
						Ref:        "ctb1",
						APIVersion: "v1alpha1",
					},
				},
				Template: v1alpha1.TriggerSpecTemplate{
					Ref:        ptr.String("tt1"),
					APIVersion: "v1alpha1",
				},
			},
		},
		builder: Trigger("name", "namespace",
			TriggerSpec(
				TriggerSpecServiceAccountName("serviceAccount"),
				TriggerSpecTemplate("tt1", "v1alpha1"),
				TriggerSpecBinding("tb1", "", "tb1", "v1alpha1"),
				TriggerSpecBinding("ctb1", "ClusterTriggerBinding", "ctb1", "v1alpha1"),
			),
		),
	}, {
		name: "One Trigger with Interceptor",
		normal: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				ServiceAccountName: "serviceAccount",
				Name:               "foo-trig",
				Interceptors: []*v1alpha1.TriggerInterceptor{{
					Webhook: &v1alpha1.WebhookInterceptor{
						ObjectRef: &corev1.ObjectReference{
							Kind:       "Service",
							Namespace:  "namespace",
							Name:       "foo",
							APIVersion: "v1",
						},
					},
				}},
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Name:       "tb1",
					Kind:       v1alpha1.NamespacedTriggerBindingKind,
					Ref:        "tb1",
					APIVersion: "v1alpha1",
				}},
				Template: v1alpha1.TriggerSpecTemplate{
					Ref:        ptr.String("tt1"),
					APIVersion: "v1alpha1",
				},
			},
		},
		builder: Trigger("name", "namespace",
			TriggerSpec(
				TriggerSpecServiceAccountName("serviceAccount"),
				TriggerSpecTemplate("tt1", "v1alpha1"),
				TriggerSpecBinding("tb1", "", "tb1", "v1alpha1"),
				TriggerSpecName("foo-trig"),
				TriggerSpecInterceptor("foo", "v1", "Service", "namespace"),
			),
		),
	}, {
		name: "One Trigger with Interceptor With Header",
		normal: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				ServiceAccountName: "serviceAccount",
				Name:               "foo-trig",
				Interceptors: []*v1alpha1.TriggerInterceptor{{
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
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Name:       "tb1",
					Kind:       v1alpha1.NamespacedTriggerBindingKind,
					Ref:        "tb1",
					APIVersion: "v1alpha1",
				}},
				Template: v1alpha1.TriggerSpecTemplate{
					Ref:        ptr.String("tt1"),
					APIVersion: "v1alpha1",
				},
			},
		},
		builder: Trigger("name", "namespace",
			TriggerSpec(
				TriggerSpecServiceAccountName("serviceAccount"),
				TriggerSpecTemplate("tt1", "v1alpha1"),
				TriggerSpecBinding("tb1", "", "tb1", "v1alpha1"),
				TriggerSpecName("foo-trig"),
				TriggerSpecInterceptor("foo", "v1", "Service", "namespace",
					TriggerSpecInterceptorParam("header1", "value1")),
			),
		),
	}, {
		name: "One Trigger with CEL Interceptor",
		normal: &v1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1alpha1.TriggerSpec{
				ServiceAccountName: "serviceAccount",
				Name:               "foo-trig",
				Interceptors: []*v1alpha1.TriggerInterceptor{{
					DeprecatedCEL: &v1alpha1.CELInterceptor{
						Filter: "body.value == 'test'",
						Overlays: []v1alpha1.CELOverlay{
							{Key: "value", Expression: "'testing'"},
						},
					},
				}},
				Bindings: []*v1alpha1.TriggerSpecBinding{{
					Name:       "tb1",
					Kind:       v1alpha1.NamespacedTriggerBindingKind,
					Ref:        "tb1",
					APIVersion: "v1alpha1",
				}},
				Template: v1alpha1.TriggerSpecTemplate{
					Ref:        ptr.String("tt1"),
					APIVersion: "v1alpha1",
				},
			},
		},
		builder: Trigger("name", "namespace",
			TriggerSpec(
				TriggerSpecServiceAccountName("serviceAccount"),
				TriggerSpecTemplate("tt1", "v1alpha1"),
				TriggerSpecBinding("tb1", "", "tb1", "v1alpha1"),
				TriggerSpecName("foo-trig"),
				TriggerSpecCELInterceptor("body.value == 'test'",
					TriggerSpecCELOverlay("value", "'testing'")),
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
