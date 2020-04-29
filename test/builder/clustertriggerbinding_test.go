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
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterTriggerBindingBuilder(t *testing.T) {
	tests := []struct {
		name    string
		normal  *v1alpha1.ClusterTriggerBinding
		builder *v1alpha1.ClusterTriggerBinding
	}{
		{
			name:    "Empty",
			normal:  &v1alpha1.ClusterTriggerBinding{},
			builder: ClusterTriggerBinding(""),
		},
		{
			name: "Name and Namespace",
			normal: &v1alpha1.ClusterTriggerBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "name",
				},
			},
			builder: ClusterTriggerBinding("name"),
		},
		{
			name: "One Param",
			normal: &v1alpha1.ClusterTriggerBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "name",
				},
				Spec: v1alpha1.TriggerBindingSpec{
					Params: []v1alpha1.Param{
						{
							Name:  "param1",
							Value: "value1",
						},
					},
				},
			},
			builder: ClusterTriggerBinding("name",
				ClusterTriggerBindingSpec(
					TriggerBindingParam("param1", "value1"),
				),
			),
		},
		{
			name: "Two Params",
			normal: &v1alpha1.ClusterTriggerBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "name",
				},
				Spec: v1alpha1.TriggerBindingSpec{
					Params: []v1alpha1.Param{
						{
							Name:  "param1",
							Value: "value1"},
						{
							Name:  "param2",
							Value: "value2",
						},
					},
				},
			},
			builder: ClusterTriggerBinding("name",
				ClusterTriggerBindingSpec(
					TriggerBindingParam("param1", "value1"),
					TriggerBindingParam("param2", "value2"),
				),
			),
		},
		{
			name: "Extra Meta",
			normal: &v1alpha1.ClusterTriggerBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "name",
					Labels: map[string]string{
						"key": "value",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterTriggerBinding",
					APIVersion: "v1alpha1",
				},
				Spec: v1alpha1.TriggerBindingSpec{},
			},
			builder: ClusterTriggerBinding("name",
				ClusterTriggerBindingMeta(
					TypeMeta("ClusterTriggerBinding", "v1alpha1"),
					Label("key", "value"),
				),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.normal, tt.builder); diff != "" {
				t.Errorf("ClusterTriggerBinding(): -want +got: %s", diff)
			}
		})
	}
}
