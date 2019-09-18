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
			name: "multiple input and output params",
			tb: bldr.TriggerBinding("name", "namespace",
				bldr.TriggerBindingSpec(
					bldr.TriggerBindingInputParam("input1", "", ""),
					bldr.TriggerBindingInputParam("input2", "", ""),
					bldr.TriggerBindingInputParam("input3", "", ""),
					bldr.TriggerBindingOutputParam("output1", "$(inputParams.input1)"),
					bldr.TriggerBindingOutputParam("output2", "$(inputParams.input2)"),
					bldr.TriggerBindingOutputParam("output3", "$(inputParams.input3)"),
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
			name: "duplicate input params",
			tb: bldr.TriggerBinding("name", "namespace",
				bldr.TriggerBindingSpec(
					bldr.TriggerBindingInputParam("input1", "", ""),
					bldr.TriggerBindingInputParam("input2", "", ""),
					bldr.TriggerBindingInputParam("input2", "", ""),
				)),
		},
		{
			name: "duplicate input params case insensitive",
			tb: bldr.TriggerBinding("name", "namespace",
				bldr.TriggerBindingSpec(
					bldr.TriggerBindingInputParam("input1", "", ""),
					bldr.TriggerBindingInputParam("input2", "", ""),
					bldr.TriggerBindingInputParam("INPUT2", "", ""),
				)),
		},
		{
			name: "duplicate output params",
			tb: bldr.TriggerBinding("name", "namespace",
				bldr.TriggerBindingSpec(
					bldr.TriggerBindingOutputParam("output1", "$(inputParams.input1)"),
					bldr.TriggerBindingOutputParam("output1", "$(inputParams.input2)"),
					bldr.TriggerBindingOutputParam("output3", "$(inputParams.input3)"),
				)),
		},
		{
			name: "duplicate output params case insensitive",
			tb: bldr.TriggerBinding("name", "namespace",
				bldr.TriggerBindingSpec(
					bldr.TriggerBindingOutputParam("output1", "$(inputParams.input1)"),
					bldr.TriggerBindingOutputParam("OUTPUT1", "$(inputParams.input2)"),
					bldr.TriggerBindingOutputParam("output3", "$(inputParams.input3)"),
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
