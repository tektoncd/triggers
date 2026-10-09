/*
Copyright 2024 The Tekton Authors

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
	"strings"

	"github.com/robfig/cron/v3"
	"github.com/tektoncd/pipeline/pkg/apis/validate"
	"k8s.io/apimachinery/pkg/api/equality"
	"knative.dev/pkg/apis"
)

// revive:disable:unused-parameter

// Validate validates a ScheduledTemplate.
func (t *ScheduledTemplate) Validate(ctx context.Context) *apis.FieldError {
	if apis.IsInDelete(ctx) {
		return nil
	}

	errs := validate.ObjectMetadata(t.GetObjectMeta()).ViaField("metadata")
	return errs.Also(t.Spec.validate(ctx).ViaField("spec"))
}

// Validate validates a ScheduledTemplateSpec.
func (s *ScheduledTemplateSpec) validate(_ctx context.Context) (errs *apis.FieldError) {
	// Check if the spec is entirely empty
	if equality.Semantic.DeepEqual(s, &ScheduledTemplateSpec{}) {
		return apis.ErrMissingField(apis.CurrentField)
	}

	hasRef := s.Ref != nil && *s.Ref != ""
	hasInline := len(s.Params) > 0 || len(s.ResourceTemplates) > 0

	switch {
	case hasRef && hasInline:
		errs = errs.Also(apis.ErrMultipleOneOf("ref", "params", "resourcetemplates"))
	case !hasRef && len(s.ResourceTemplates) == 0:
		// Keep requiring resourcetemplates when not using ref (same as today / TriggerTemplate).
		errs = errs.Also(apis.ErrMissingOneOf("ref", "resourcetemplates"))
	case hasRef:
		// ref path: no inline template validation
	default:
		errs = errs.Also(validateResourceTemplates(s.ResourceTemplates).ViaField("resourcetemplates"))
		errs = errs.Also(verifyParamDeclarations(s.Params, s.ResourceTemplates).ViaField("resourcetemplates"))
	}
	//  Validate Schedule
	if s.Schedule == "" {
		errs = errs.Also(apis.ErrMissingField("schedule"))
	} else if _, err := cron.ParseStandard(s.Schedule); err != nil {
		errs = errs.Also(apis.ErrInvalidValue(err, "schedule"))
	}
	// Validate CloudEventSink (if necessary)
	if s.CloudEventSink != nil {
		if s.CloudEventSink.Host == "" {
			errs = errs.Also(apis.ErrMissingField("cloudEventSink.host"))
		}
		scheme := strings.ToLower(s.CloudEventSink.Scheme)
		if scheme == "" {
			errs = errs.Also(apis.ErrMissingField("cloudEventSink.scheme"))
		} else if scheme != "http" && scheme != "https" {
			errs = errs.Also(apis.ErrInvalidValue(s.CloudEventSink.Scheme, "cloudEventSink.scheme"))
		}
	}

	return errs
}
