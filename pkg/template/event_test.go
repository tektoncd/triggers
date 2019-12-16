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

package template

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/test"
	bldr "github.com/tektoncd/triggers/test/builder"
	"k8s.io/apimachinery/pkg/util/rand"
)

// TODO(#252): Split testcases from NewResourcesTests into TestResolveParams and TestResolveResources
func Test_NewResources(t *testing.T) {
	type args struct {
		body    []byte
		header  map[string][]string
		binding ResolvedTrigger
	}
	tests := []struct {
		name string
		args args
		want []json.RawMessage
	}{{
		name: "empty",
		args: args{
			body:   json.RawMessage{},
			header: map[string][]string{},
			binding: ResolvedTrigger{
				TriggerTemplate: bldr.TriggerTemplate("tt", "namespace"),
				TriggerBindings: []*triggersv1.TriggerBinding{bldr.TriggerBinding("tb", "namespace")},
			},
		},
		want: []json.RawMessage{},
	}, {
		name: "one resource template",
		args: args{
			body:   json.RawMessage(`{"foo": "bar"}`),
			header: map[string][]string{"one": {"1"}},
			binding: ResolvedTrigger{
				TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
					bldr.TriggerTemplateSpec(
						bldr.TriggerTemplateParam("param1", "description", ""),
						bldr.TriggerTemplateParam("param2", "description", ""),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt1": "$(params.param1)-$(params.param2)"}`)),
					),
				),
				TriggerBindings: []*triggersv1.TriggerBinding{
					bldr.TriggerBinding("tb", "namespace",
						bldr.TriggerBindingSpec(
							bldr.TriggerBindingParam("param1", "$(body.foo)"),
							bldr.TriggerBindingParam("param2", "$(header.one)"),
						),
					),
				},
			},
		},
		want: []json.RawMessage{
			json.RawMessage(`{"rt1": "bar-1"}`),
		},
	}, {
		name: "multiple resource templates",
		args: args{
			body:   json.RawMessage(`{"foo": "bar"}`),
			header: map[string][]string{"one": {"1"}},
			binding: ResolvedTrigger{
				TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
					bldr.TriggerTemplateSpec(
						bldr.TriggerTemplateParam("param1", "description", ""),
						bldr.TriggerTemplateParam("param2", "description", ""),
						bldr.TriggerTemplateParam("param3", "description", "default2"),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt1": "$(params.param1)-$(params.param2)"}`)),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt2": "$(params.param3)"}`)),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt3": "rt3"}`)),
					),
				),
				TriggerBindings: []*triggersv1.TriggerBinding{
					bldr.TriggerBinding("tb", "namespace",
						bldr.TriggerBindingSpec(
							bldr.TriggerBindingParam("param1", "$(body.foo)"),
							bldr.TriggerBindingParam("param2", "$(header.one)"),
						),
					),
				},
			},
		},
		want: []json.RawMessage{
			json.RawMessage(`{"rt1": "bar-1"}`),
			json.RawMessage(`{"rt2": "default2"}`),
			json.RawMessage(`{"rt3": "rt3"}`),
		},
	}, {
		name: "one resource template with one uid",
		args: args{
			body: json.RawMessage(`{"foo": "bar"}`),
			binding: ResolvedTrigger{
				TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
					bldr.TriggerTemplateSpec(
						bldr.TriggerTemplateParam("param1", "description", ""),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt1": "$(params.param1)-$(uid)"}`)),
					),
				),
				TriggerBindings: []*triggersv1.TriggerBinding{
					bldr.TriggerBinding("tb", "namespace",
						bldr.TriggerBindingSpec(
							bldr.TriggerBindingParam("param1", "$(body.foo)"),
						),
					),
				},
			},
		},
		want: []json.RawMessage{
			json.RawMessage(`{"rt1": "bar-cbhtc"}`),
		},
	}, {
		name: "one resource template with three uid",
		args: args{
			body: json.RawMessage(`{"foo": "bar"}`),
			binding: ResolvedTrigger{
				TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
					bldr.TriggerTemplateSpec(
						bldr.TriggerTemplateParam("param1", "description", ""),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt1": "$(params.param1)-$(uid)-$(uid)", "rt2": "$(uid)"}`)),
					),
				),
				TriggerBindings: []*triggersv1.TriggerBinding{
					bldr.TriggerBinding("tb", "namespace",
						bldr.TriggerBindingSpec(
							bldr.TriggerBindingParam("param1", "$(body.foo)"),
						),
					),
				},
			},
		},
		want: []json.RawMessage{
			json.RawMessage(`{"rt1": "bar-cbhtc-cbhtc", "rt2": "cbhtc"}`),
		},
	}, {
		name: "multiple resource templates with multiple uid",
		args: args{
			body: json.RawMessage(`{"foo": "bar"}`),
			binding: ResolvedTrigger{
				TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
					bldr.TriggerTemplateSpec(
						bldr.TriggerTemplateParam("param1", "description", ""),
						bldr.TriggerTemplateParam("param2", "description", "default2"),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt1": "$(params.param1)-$(uid)", "$(uid)": "$(uid)"}`)),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt2": "$(params.param2)-$(uid)"}`)),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt3": "rt3"}`)),
					),
				),
				TriggerBindings: []*triggersv1.TriggerBinding{
					bldr.TriggerBinding("tb", "namespace",
						bldr.TriggerBindingSpec(
							bldr.TriggerBindingParam("param1", "$(body.foo)"),
						),
					),
				},
			},
		},
		want: []json.RawMessage{
			json.RawMessage(`{"rt1": "bar-cbhtc", "cbhtc": "cbhtc"}`),
			json.RawMessage(`{"rt2": "default2-bsvjp"}`),
			json.RawMessage(`{"rt3": "rt3"}`),
		},
	}, {
		name: "one resource template multiple bindings",
		args: args{
			body:   json.RawMessage(`{"foo": "bar"}`),
			header: map[string][]string{"one": {"1"}},
			binding: ResolvedTrigger{
				TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
					bldr.TriggerTemplateSpec(
						bldr.TriggerTemplateParam("param1", "description", ""),
						bldr.TriggerTemplateParam("param2", "description", ""),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt1": "$(params.param1)-$(params.param2)"}`)),
					),
				),
				TriggerBindings: []*triggersv1.TriggerBinding{
					bldr.TriggerBinding("tb", "namespace",
						bldr.TriggerBindingSpec(
							bldr.TriggerBindingParam("param1", "$(body.foo)"),
						),
					),
					bldr.TriggerBinding("tb2", "namespace",
						bldr.TriggerBindingSpec(
							bldr.TriggerBindingParam("param2", "$(header.one)"),
						),
					),
				},
			},
		},
		want: []json.RawMessage{
			json.RawMessage(`{"rt1": "bar-1"}`),
		},
	}, {
		name: "bindings with static values",
		args: args{
			body: json.RawMessage(`{"foo": "bar"}`),
			binding: ResolvedTrigger{
				TriggerTemplate: bldr.TriggerTemplate("tt", "ns", bldr.TriggerTemplateSpec(
					bldr.TriggerTemplateParam("p1", "", ""),
					bldr.TriggerTemplateParam("p2", "", ""),
					bldr.TriggerResourceTemplate(json.RawMessage(`{"p1": "$(params.p1)", "p2": "$(params.p2)"}`)),
				),
				),
				TriggerBindings: []*triggersv1.TriggerBinding{
					bldr.TriggerBinding("tb", "ns", bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("p1", "static_value"),
						bldr.TriggerBindingParam("p2", "$(body.foo)"),
					)),
				},
			},
		},
		want: []json.RawMessage{
			json.RawMessage(`{"p1": "static_value", "p2": "bar"}`),
		},
	}, {
		name: "bindings with combination of static values ",
		args: args{
			body: json.RawMessage(`{"foo": "fooValue", "bar": "barValue"}`),
			binding: ResolvedTrigger{
				TriggerTemplate: bldr.TriggerTemplate("tt", "ns", bldr.TriggerTemplateSpec(
					bldr.TriggerTemplateParam("p1", "", ""),
					bldr.TriggerResourceTemplate(json.RawMessage(`{"p1": "$(params.p1)"`)),
				),
				),
				TriggerBindings: []*triggersv1.TriggerBinding{
					bldr.TriggerBinding("tb", "ns", bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("p1", "Event values are - foo: $(body.foo); bar: $(body.bar)"),
					)),
				},
			},
		},
		want: []json.RawMessage{
			json.RawMessage(`{"p1": "Event values are - foo: fooValue; bar: barValue"`),
		},
	}, {
		name: "event value is JSON string",
		args: args{
			body: json.RawMessage(`{"a": "b"}`),
			binding: ResolvedTrigger{
				TriggerTemplate: bldr.TriggerTemplate("tt", "ns", bldr.TriggerTemplateSpec(
					bldr.TriggerTemplateParam("p1", "", ""),
					bldr.TriggerResourceTemplate(json.RawMessage(`{"p1": "$(params.p1)"}`)),
				),
				),
				TriggerBindings: []*triggersv1.TriggerBinding{
					bldr.TriggerBinding("tb", "ns", bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("p1", "$(body)"),
					)),
				},
			},
		},
		want: []json.RawMessage{
			json.RawMessage(`{"p1": "{\"a\":\"b\"}"}`),
		},
	}, {
		name: "header event values",
		args: args{
			header: map[string][]string{
				"a": {"singlevalue"},
				"b": {"multiple", "values"},
			},
			binding: ResolvedTrigger{
				TriggerTemplate: bldr.TriggerTemplate("tt", "ns", bldr.TriggerTemplateSpec(
					bldr.TriggerTemplateParam("p1", "", ""),
					bldr.TriggerTemplateParam("p2", "", ""),
					bldr.TriggerResourceTemplate(json.RawMessage(`{"p1": "$(params.p1)","p2": "$(params.p2)"}`)),
				),
				),
				TriggerBindings: []*triggersv1.TriggerBinding{
					bldr.TriggerBinding("tb", "ns", bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("p1", "$(header.a)"),
						bldr.TriggerBindingParam("p2", "$(header.b)"),
					)),
				},
			},
		},
		want: []json.RawMessage{
			json.RawMessage(`{"p1": "singlevalue","p2": "multiple,values"}`),
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This seeds Uid() to return 'cbhtc'
			rand.Seed(0)
			params, err := ResolveParams(tt.args.binding.TriggerBindings, tt.args.body, tt.args.header, tt.args.binding.TriggerTemplate.Spec.Params)
			if err != nil {
				t.Fatalf("ResolveParams() returned unexpected error: %s", err)
			}
			got := ResolveResources(tt.args.binding.TriggerTemplate, params)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				stringDiff := cmp.Diff(convertJSONRawMessagesToString(tt.want), convertJSONRawMessagesToString(got))
				t.Errorf("ResolveResources(): -want +got: %s", stringDiff)
			}
		})
	}
}

func TestResolveParams(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		params   []pipelinev1.ParamSpec
		bindings []*triggersv1.TriggerBinding
		want     []pipelinev1.Param
	}{{
		name: "values with newlines",
		body: json.RawMessage(`{"foo": "bar\r\nbaz"}`),
		params: []pipelinev1.ParamSpec{{
			Name: "param1",
		}, {
			Name: "param2",
		}},
		bindings: []*triggersv1.TriggerBinding{
			bldr.TriggerBinding("tb", "namespace",
				bldr.TriggerBindingSpec(
					bldr.TriggerBindingParam("param1", "qux"),
					bldr.TriggerBindingParam("param2", "$(body.foo)"),
				),
			),
		},
		want: []pipelinev1.Param{{
			Name: "param1",
			Value: pipelinev1.ArrayOrString{
				StringVal: "qux",
				Type:      pipelinev1.ParamTypeString,
			},
		}, {
			Name: "param2",
			Value: pipelinev1.ArrayOrString{
				StringVal: "bar\\r\\nbaz",
				Type:      pipelinev1.ParamTypeString,
			},
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := ResolveParams(tt.bindings, tt.body, map[string][]string{}, tt.params)
			if err != nil {
				t.Fatalf("ResolveParams() returned unexpected error: %s", err)
			}
			if diff := cmp.Diff(tt.want, params, cmpopts.SortSlices(test.CompareParams)); diff != "" {
				t.Errorf("didn't get expected params -want + got: %s", diff)
			}
		})
	}
}

func convertJSONRawMessagesToString(rawMessages []json.RawMessage) []string {
	stringMessages := make([]string, len(rawMessages))
	for i := range rawMessages {
		stringMessages[i] = string(rawMessages[i])
	}
	return stringMessages
}

func Test_NewResources_error(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		header   map[string][]string
		elParams []pipelinev1.Param
		binding  ResolvedTrigger
	}{{
		name: "bodypath not found in body",
		body: json.RawMessage(`{"foo": "bar"}`),
		binding: ResolvedTrigger{
			TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
				bldr.TriggerTemplateSpec(
					bldr.TriggerTemplateParam("param1", "description", ""),
				),
			),
			TriggerBindings: []*triggersv1.TriggerBinding{
				bldr.TriggerBinding("tb", "namespace",
					bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("param1", "$(body.bogusvalue)"),
					),
				),
			},
		},
	}, {
		name:   "header not found in event",
		body:   json.RawMessage(`{"foo": "bar"}`),
		header: map[string][]string{"One": {"one"}},
		binding: ResolvedTrigger{
			TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
				bldr.TriggerTemplateSpec(
					bldr.TriggerTemplateParam("param1", "description", ""),
				),
			),
			TriggerBindings: []*triggersv1.TriggerBinding{
				bldr.TriggerBinding("tb", "namespace",
					bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("param1", "$(header.bogusvalue)"),
					),
				),
			},
		},
	}, {
		name: "merge params error",
		elParams: []pipelinev1.Param{
			{
				Name:  "param1",
				Value: pipelinev1.ArrayOrString{StringVal: "value1", Type: pipelinev1.ParamTypeString},
			},
		},
		binding: ResolvedTrigger{
			TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
				bldr.TriggerTemplateSpec(
					bldr.TriggerTemplateParam("param1", "description", ""),
				),
			),
			TriggerBindings: []*triggersv1.TriggerBinding{
				bldr.TriggerBinding("tb", "namespace",
					bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("param1", "$(body.bogusvalue)"),
					),
				),
			},
		},
	}, {
		name: "conflicting bindings",
		binding: ResolvedTrigger{
			TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
				bldr.TriggerTemplateSpec(
					bldr.TriggerTemplateParam("param1", "description", ""),
				),
			),
			TriggerBindings: []*triggersv1.TriggerBinding{
				bldr.TriggerBinding("tb", "namespace",
					bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("param1", "foo"),
					),
				),
				bldr.TriggerBinding("tb2", "namespace",
					bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("param1", "bar"),
					),
				),
			},
		},
	},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveParams(tt.binding.TriggerBindings, tt.body, tt.header, tt.binding.TriggerTemplate.Spec.Params)
			if err == nil {
				t.Errorf("NewResources() did not return error when expected; got: %s", got)
			}
		})
	}
}
