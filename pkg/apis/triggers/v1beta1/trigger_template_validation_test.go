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

package v1beta1_test

import (
	"context"
	"testing"

	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	b "github.com/tektoncd/triggers/test/builder"

	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
)

func simpleResourceTemplate(t *testing.T) runtime.RawExtension {
	return test.RawExtension(t, pipelinev1alpha1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "PipelineRun",
		},
	})
}

func v1beta1ResourceTemplate(t *testing.T) runtime.RawExtension {
	return test.RawExtension(t, pipelinev1beta1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "PipelineRun",
		},
	})
}

func paramResourceTemplate(t *testing.T) runtime.RawExtension {
	return test.RawExtension(t, pipelinev1alpha1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "PipelineRun",
		},
		Spec: pipelinev1alpha1.PipelineRunSpec{
			Params: []pipelinev1alpha1.Param{
				{
					Name: "message",
					Value: pipelinev1alpha1.ArrayOrString{
						Type:      pipelinev1alpha1.ParamTypeString,
						StringVal: "$(tt.params.foo)",
					},
				},
			},
		},
	})
}

func invalidParamResourceTemplate(t *testing.T) runtime.RawExtension {
	return test.RawExtension(t, pipelinev1alpha1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "PipelineRun",
		},
		Spec: pipelinev1alpha1.PipelineRunSpec{
			Params: []pipelinev1alpha1.Param{
				{
					Name: "message",
					Value: pipelinev1alpha1.ArrayOrString{
						Type:      pipelinev1alpha1.ParamTypeString,
						StringVal: "$(.foo)",
					},
				},
			},
		},
	})
}

func TestTriggerTemplate_Validate(t *testing.T) {
	tcs := []struct {
		name     string
		template *v1beta1.TriggerTemplate
		want     *apis.FieldError
	}{
		{
			name: "invalid objectmetadata, name with dot",
			template: b.TriggerTemplate("t.t", "foo", b.TriggerTemplateSpec(
				b.TriggerTemplateParam("foo", "desc", "val"),
				b.TriggerResourceTemplate(simpleResourceTemplate(t)))),
			want: &apis.FieldError{
				Message: "Invalid resource name: special character . must not be present",
				Paths:   []string{"metadata.name"},
			},
		},
		{
			name: "invalid objectmetadata, name too long",
			template: b.TriggerTemplate(
				"ttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttt",
				"foo", b.TriggerTemplateSpec(
					b.TriggerTemplateParam("foo", "desc", "val"),
					b.TriggerResourceTemplate(simpleResourceTemplate(t)))),
			want: &apis.FieldError{
				Message: "Invalid resource name: length must be no more than 63 characters",
				Paths:   []string{"metadata.name"},
			},
		},
		{
			name: "valid template",
			template: b.TriggerTemplate("tt", "foo", b.TriggerTemplateSpec(
				b.TriggerTemplateParam("foo", "desc", "val"),
				b.TriggerResourceTemplate(simpleResourceTemplate(t)))),
			want: nil,
		}, {
			name: "valid v1beta1 template",
			template: b.TriggerTemplate("tt", "foo", b.TriggerTemplateSpec(
				b.TriggerTemplateParam("foo", "desc", "val"),
				b.TriggerResourceTemplate(v1beta1ResourceTemplate(t)))),
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
				b.TriggerResourceTemplate(test.RawExtension(t, pipelinev1alpha1.PipelineRun{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "tekton.dev/v1beta1",
					},
				})))),
			want: &apis.FieldError{
				Message: "missing field(s)",
				Paths:   []string{"spec.resourcetemplates[0].kind"},
			},
		}, {
			name: "resource template missing apiVersion",
			template: b.TriggerTemplate("tt", "foo", b.TriggerTemplateSpec(
				b.TriggerTemplateParam("foo", "desc", "val"),
				b.TriggerResourceTemplate(test.RawExtension(t, pipelinev1alpha1.PipelineRun{
					TypeMeta: metav1.TypeMeta{
						Kind: "PipelineRun",
					},
				})))),
			want: &apis.FieldError{
				Message: "missing field(s)",
				Paths:   []string{"spec.resourcetemplates[0].apiVersion"},
			},
		}, {
			name: "resource template invalid apiVersion",
			template: b.TriggerTemplate("tt", "foo", b.TriggerTemplateSpec(
				b.TriggerTemplateParam("foo", "desc", "val"),
				b.TriggerResourceTemplate(test.RawExtension(t, pipelinev1alpha1.PipelineRun{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "foobar",
						Kind:       "pipelinerun",
					},
				})))),
			want: &apis.FieldError{
				Message: `invalid value: no kind "pipelinerun" is registered for version "foobar"`,
				Paths:   []string{"spec.resourcetemplates[0]"},
			},
		}, {
			name: "resource template invalid kind",
			template: b.TriggerTemplate("tt", "foo", b.TriggerTemplateSpec(
				b.TriggerTemplateParam("foo", "desc", "val"),
				b.TriggerResourceTemplate(test.RawExtension(t, pipelinev1alpha1.PipelineRun{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "foo",
						Kind:       "tekton.dev/v1beta1",
					},
				})))),
			want: &apis.FieldError{
				Message: `invalid value: no kind "tekton.dev/v1beta1" is registered for version "foo"`,
				Paths:   []string{"spec.resourcetemplates[0]"},
			},
		}, {
			name: "tt.params used in resource template are declared",
			template: b.TriggerTemplate("tt", "foo", b.TriggerTemplateSpec(
				b.TriggerTemplateParam("foo", "desc", "val"),
				b.TriggerResourceTemplate(paramResourceTemplate(t)))),
			want: nil,
		}, {
			name: "tt.params used in resource template are not declared",
			template: b.TriggerTemplate("tt", "foo", b.TriggerTemplateSpec(
				b.TriggerResourceTemplate(paramResourceTemplate(t)))),
			want: &apis.FieldError{
				Message: "invalid value: undeclared param '$(tt.params.foo)'",
				Paths:   []string{"spec.resourcetemplates[0]"},
				Details: "'$(tt.params.foo)' must be declared in spec.params",
			},
		}, {
			name: "invalid params used in resource template are not declared",
			template: b.TriggerTemplate("tt", "foo", b.TriggerTemplateSpec(
				b.TriggerResourceTemplate(invalidParamResourceTemplate(t)))),
			want: nil,
		}, {
			name: "invalid params used in resource template are declared",
			template: b.TriggerTemplate("tt", "foo", b.TriggerTemplateSpec(
				b.TriggerTemplateParam("foo", "desc", "val"),
				b.TriggerResourceTemplate(invalidParamResourceTemplate(t)))),
			want: nil,
		}, {
			name:     "no spec to triggertemplate",
			template: b.TriggerTemplate("tt", "foo", b.TriggerTemplateSpec()),
			want:     apis.ErrMissingField("spec", "spec.resourcetemplates"),
		}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.template.Validate(context.Background())
			if d := cmp.Diff(got.Error(), tc.want.Error()); d != "" {
				t.Errorf("TriggerTemplate Validation failed: %s", d)
			}
		})
	}
}
