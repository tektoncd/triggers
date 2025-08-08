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
	"knative.dev/pkg/apis"
)

const (
	PayloadValidationAnnotation = "tekton.dev/payload-validation"
)

func ValidateAnnotations(annotations map[string]string) *apis.FieldError {
	var errs *apis.FieldError

	if value, ok := annotations[PayloadValidationAnnotation]; ok {
		if value != "true" && value != "false" {
			errs = errs.Also(apis.ErrInvalidValue(PayloadValidationAnnotation+" annotation must have value 'true' or 'false'", "metadata.annotations"))
		}
	}

	return errs
}
