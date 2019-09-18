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

package v1alpha1

import (
	"context"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"knative.dev/pkg/apis"
	"strings"
)

// Validate TriggerBinding.
func (t *TriggerBinding) Validate(ctx context.Context) *apis.FieldError {
	return t.Spec.Validate(ctx)
}

// Validate TriggerBindingSpec.
func (s *TriggerBindingSpec) Validate(ctx context.Context) *apis.FieldError {
	if err := validateInputParams(s.InputParams); err != nil {
		return err
	}
	if err := validateOutputParams(s.OutputParams); err != nil {
		return err
	}
	return nil
}

func validateInputParams(params []v1alpha1.ParamSpec) *apis.FieldError {
	paramNames := make([]string, len(params))
	for i, param := range params {
		paramNames[i] = param.Name
	}
	if duplicateStrings(paramNames) {
		return apis.ErrMultipleOneOf("spec.inputParams")
	}
	return nil
}

func validateOutputParams(params []v1alpha1.Param) *apis.FieldError {
	paramNames := make([]string, len(params))
	for i, param := range params {
		paramNames[i] = param.Name
	}
	if duplicateStrings(paramNames) {
		return apis.ErrMultipleOneOf("spec.outputParams")
	}
	return nil
}

// Return true if there are duplicate strings in the list. Converts strings
// to lowercase before comparing.
func duplicateStrings(list []string) bool {
	seen := map[string]struct{}{}
	for _, s := range list {
		if _, ok := seen[strings.ToLower(s)]; ok {
			return true
		}
		seen[s] = struct{}{}
	}
	return false
}
