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
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	b "github.com/tektoncd/triggers/test/builder"
	"knative.dev/pkg/apis"
)

var simpleResourceTemplate = json.RawMessage(`{ "apiVersion": "foobar", "kind": "foo"}`)

func TestTriggerTemplateSpec_Validate(t *testing.T) {
	tcs := []struct {
		name     string
		template *v1alpha1.TriggerTemplate
		want     *apis.FieldError
	}{{
		name: "valid template",
		template: b.TriggerTemplate("tt", "foo", b.TriggerTemplateSpec(
			b.TriggerTemplateParam("foo", "desc", "val"),
			b.TriggerResourceTemplate(simpleResourceTemplate))),
		want: nil,
	}, {
		name: "missing resource template",
		template: b.TriggerTemplate("tt", "foo", b.TriggerTemplateSpec(
			b.TriggerTemplateParam("foo", "desc", "val"))),
		want: &apis.FieldError{
			Message: "missing field(s)",
			Paths:   []string{"spec.resourcetemplates"},
		},
	}, {
		name: "resource template missing kind",
		template: b.TriggerTemplate("tt", "foo", b.TriggerTemplateSpec(
			b.TriggerTemplateParam("foo", "desc", "val"),
			b.TriggerResourceTemplate(json.RawMessage(`{"apiVersion": "foo"}`)))),
		want: &apis.FieldError{
			Message: "missing field(s)",
			Paths:   []string{"spec.resourcetemplates[0].kind"},
		},
	}, {
		name: "resource template missing apiVersion",
		template: b.TriggerTemplate("tt", "foo", b.TriggerTemplateSpec(
			b.TriggerTemplateParam("foo", "desc", "val"),
			b.TriggerResourceTemplate(json.RawMessage(`{"kind": "foo"}`)))),
		want: &apis.FieldError{
			Message: "missing field(s)",
			Paths:   []string{"spec.resourcetemplates[0].apiVersion"},
		},
	}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.template.Validate(context.Background())
			if d := cmp.Diff(got, tc.want, cmpopts.IgnoreUnexported(apis.FieldError{})); d != "" {
				t.Errorf("TriggerTemplate Validation failed: %s", d)
			}
		})
	}
}
