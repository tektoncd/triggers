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

	"github.com/google/go-cmp/cmp"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

func TestTriggerTemplateSetDefaults(t *testing.T) {
	tests := []struct {
		name string
		in   *v1alpha1.TriggerTemplate
		want *v1alpha1.TriggerTemplate
	}{{
		name: "empty params defaults to params of type to string",
		in: &v1alpha1.TriggerTemplate{
			Spec: v1alpha1.TriggerTemplateSpec{
				Params: []pipelinev1beta1.ParamSpec{{}},
			},
		},
		want: &v1alpha1.TriggerTemplate{
			Spec: v1alpha1.TriggerTemplateSpec{
				Params: []pipelinev1beta1.ParamSpec{{
					Type: pipelinev1beta1.ParamTypeString,
				}},
			},
		},
	}, {
		name: "params of type array",
		in: &v1alpha1.TriggerTemplate{
			Spec: v1alpha1.TriggerTemplateSpec{
				Params: []pipelinev1beta1.ParamSpec{{
					Name:        "contenttype",
					Type:        pipelinev1beta1.ParamTypeArray,
					Description: "The Content-Type of the event",
				}},
			},
		},
		want: &v1alpha1.TriggerTemplate{
			Spec: v1alpha1.TriggerTemplateSpec{
				Params: []pipelinev1beta1.ParamSpec{{
					Name:        "contenttype",
					Type:        pipelinev1beta1.ParamTypeArray,
					Description: "The Content-Type of the event",
				}},
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in
			got.SetDefaults(context.Background())
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("SetDefaults (-want, +got) = %v", diff)
			}
		})
	}
}
