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
	bldr "github.com/tektoncd/triggers/test/builder"
)

func Test_TriggerBindingValidate(t *testing.T) {
	tests := []struct {
		name string
		tb   *v1alpha1.TriggerBinding
	}{
		{
			name: "empty",
			tb:   bldr.TriggerBinding("name", "namespace"),
		},
		{
			name: "multiple params",
			tb: bldr.TriggerBinding("name", "namespace",
				bldr.TriggerBindingSpec(
					bldr.TriggerBindingParam("param1", "$(event.input1)"),
					bldr.TriggerBindingParam("param2", "$(event.input2)"),
					bldr.TriggerBindingParam("param3", "$(event.input3)"),
				)),
		},
		{
			name: "multiple params case sensitive",
			tb: bldr.TriggerBinding("name", "namespace",
				bldr.TriggerBindingSpec(
					bldr.TriggerBindingParam("param1", "$(event.input1)"),
					bldr.TriggerBindingParam("PARAM1", "$(event.input2)"),
					bldr.TriggerBindingParam("Param1", "$(event.input3)"),
				)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.tb.Validate(context.Background()); err != nil {
				t.Errorf("TriggerBinding.Validate() returned error: %s", err)
			}
		})
	}
}

func Test_TriggerBindingValidate_error(t *testing.T) {
	tests := []struct {
		name string
		tb   *v1alpha1.TriggerBinding
	}{
		{
			name: "duplicate params",
			tb: bldr.TriggerBinding("name", "namespace",
				bldr.TriggerBindingSpec(
					bldr.TriggerBindingParam("param1", "$(event.param1)"),
					bldr.TriggerBindingParam("param1", "$(event.param1)"),
					bldr.TriggerBindingParam("param3", "$(event.param1)"),
				)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.tb.Validate(context.Background()); err == nil {
				t.Errorf("TriggerBinding.Validate() expected error for TriggerBinding: %v", tt.tb)
			}
		})
	}
}
