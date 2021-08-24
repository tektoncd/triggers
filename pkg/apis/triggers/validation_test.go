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

package triggers

import (
	"testing"
)

func Test_PayloadValidationAnnotation_Valid(t *testing.T) {
	annotations := map[string]string{PayloadValidationAnnotation: "false"}
	err := ValidateAnnotations(annotations)
	if err != nil {
		t.Errorf("Unexpected Error: %v", err)
	}
}

func Test_PayloadValidationAnnotation_InvalidValue(t *testing.T) {
	annotations := map[string]string{PayloadValidationAnnotation: "abc"}
	err := ValidateAnnotations(annotations)
	if err == nil {
		t.Errorf("Expected Error but got nil")
	}
}
