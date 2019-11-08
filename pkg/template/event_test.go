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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	bldr "github.com/tektoncd/triggers/test/builder"
	"golang.org/x/xerrors"

	"k8s.io/apimachinery/pkg/util/rand"
)

func Test_BodyPathVarRegex(t *testing.T) {
	tests := []string{
		"$(body)",
		"$(body.a-b)",
		"$(body.a1)",
		"$(body.a.b)",
		"$(body.a.b.c)",
		"$(body.1.b.c\\.e/f)",
		"$(body.#(a==b))",
		"$(body.#(a>1)#)",
		"$(body.#(a%\"D*\")#.c)",
		"$(body.#(a!%\"D*\").c)",
	}
	for _, bodyPathVar := range tests {
		t.Run(bodyPathVar, func(t *testing.T) {
			if !bodyPathVarRegex.MatchString(bodyPathVar) {
				t.Errorf("bodyPathVarRegex.MatchString(%s) = false, want = true", bodyPathVar)
			}
		})
	}
}

func Test_BodyPathVarRegex_invalid(t *testing.T) {
	tests := []string{
		"$body",
		"$[body]",
		"${body}",
		"$(body.)",
		"$(body.@)",
		"$(body.$a)",
		"$(body#a)",
		"$(body@#)",
		"body.a",
		"body",
		"${{body}",
		"${body",
	}
	for _, bodyPathVar := range tests {
		t.Run(bodyPathVar, func(t *testing.T) {
			if bodyPathVarRegex.MatchString(bodyPathVar) {
				t.Errorf("bodyPathVarRegex.MatchString(%s) = true, want = false", bodyPathVar)
			}
		})
	}
}

func Test_HeaderVarRegex(t *testing.T) {
	tests := []string{
		"$(header)",
		"$(header.a-b)",
		"$(header.a1)",
	}
	for _, headerVar := range tests {
		t.Run(headerVar, func(t *testing.T) {
			if !headerVarRegex.MatchString(headerVar) {
				t.Errorf("headerVarRegex.MatchString(%s) = false, want = true", headerVar)
			}
		})
	}
}

func Test_HeaderVarRegex_invalid(t *testing.T) {
	tests := []string{
		"$(header.a.b)",
		"$(header.a.b.c)",
		"$header",
		"$[header]",
		"${header}",
		"$(header.)",
		"$(header..)",
		"$(header.$a)",
		"header.a",
		"header",
		"${{header}",
		"${header",
	}
	for _, headerVar := range tests {
		t.Run(headerVar, func(t *testing.T) {
			if headerVarRegex.MatchString(headerVar) {
				t.Errorf("headerVarRegex.MatchString(%s) = true, want = false", headerVar)
			}
		})
	}
}

func Test_GetBodyPathFromVar(t *testing.T) {
	tests := []struct {
		bodyPathVar string
		want        string
	}{
		{bodyPathVar: "$(body)", want: ""},
		{bodyPathVar: "$(body.a-b)", want: "a-b"},
		{bodyPathVar: "$(body.a1)", want: "a1"},
		{bodyPathVar: "$(body.a.b)", want: "a.b"},
		{bodyPathVar: "$(body.a.b.c)", want: "a.b.c"},
	}
	for _, tt := range tests {
		t.Run(tt.bodyPathVar, func(t *testing.T) {
			if bodyPath := getBodyPathFromVar(tt.bodyPathVar); bodyPath != tt.want {
				t.Errorf("getBodyPathFromVar() = %s, want = %s", bodyPath, tt.want)
			}
		})
	}
}

func Test_GetHeaderFromVar(t *testing.T) {
	tests := []struct {
		headerVar string
		want      string
	}{
		{headerVar: "$(header)", want: ""},
		{headerVar: "$(header.a-b)", want: "a-b"},
		{headerVar: "$(header.a1)", want: "a1"},
		{headerVar: "$(header.a.b)", want: "a.b"},
	}
	for _, tt := range tests {
		t.Run(tt.headerVar, func(t *testing.T) {
			if header := getHeaderFromVar(tt.headerVar); header != tt.want {
				t.Errorf("getHeaderFromVar() = %s, want = %s", header, tt.want)
			}
		})
	}
}

func Test_getBodyPathValue(t *testing.T) {
	body := `{"empty": "", "null": null, "one": "one", "two": {"two": "twovalue"}, "three": {"three": {"three": {"three": {"three": "threevalue"}}}}}`
	bodyJSON := json.RawMessage(body)
	type args struct {
		body     []byte
		bodyPath string
	}
	tests := []struct {
		args args
		want string
	}{
		{
			args: args{
				body:     bodyJSON,
				bodyPath: "",
			},
			want: strings.Replace(body, `"`, `\"`, -1),
		},
		{
			args: args{
				body:     bodyJSON,
				bodyPath: "one",
			},
			want: "one",
		},
		{
			args: args{
				body:     bodyJSON,
				bodyPath: "two",
			},
			want: `{\"two\": \"twovalue\"}`,
		},
		{
			args: args{
				body:     bodyJSON,
				bodyPath: "three.three.three.three.three",
			},
			want: "threevalue",
		},
		{
			args: args{
				body:     bodyJSON,
				bodyPath: "empty",
			},
			want: "",
		},
		{
			args: args{
				body:     bodyJSON,
				bodyPath: "null",
			},
			want: "null",
		},
	}
	for _, tt := range tests {
		t.Run(tt.args.bodyPath, func(t *testing.T) {
			got, err := getBodyPathValue(tt.args.body, tt.args.bodyPath)
			if err != nil {
				t.Errorf("getBodyPathValue() error: %s", err)
			} else if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("getBodyPathValue(): -want +got: %s", diff)
			}
		})
	}
}

func Test_getBodyPathValue_error(t *testing.T) {
	bodyJSON := json.RawMessage(`{"one": "onevalue", "two": {"two": "twovalue"}, "three": {"three": {"three": {"three": {"three": "threevalue"}}}}}`)
	tests := []struct {
		body     []byte
		bodyPath string
	}{
		{
			body:     bodyJSON,
			bodyPath: "boguspath",
		},
		{
			body:     bodyJSON,
			bodyPath: "two.bogus",
		},
		{
			body:     bodyJSON,
			bodyPath: "three.three.bogus.three",
		},
	}
	for _, tt := range tests {
		t.Run(tt.bodyPath, func(t *testing.T) {
			got, err := getBodyPathValue(tt.body, tt.bodyPath)
			if err == nil {
				t.Errorf("getBodyPathValue() did not return error when expected; got: %s", got)
			}
		})
	}
}

func Test_getHeaderValue(t *testing.T) {
	header := map[string][]string{"one": {"one"}, "two": {"one", "two"}, "three": {"one", "two", "three"}}
	type args struct {
		header     map[string][]string
		headerName string
	}
	tests := []struct {
		args args
		want string
	}{
		{
			args: args{
				header:     header,
				headerName: "",
			},
			want: `{\"one\":[\"one\"],\"three\":[\"one\",\"two\",\"three\"],\"two\":[\"one\",\"two\"]}`,
		},
		{
			args: args{
				header:     header,
				headerName: "one",
			},
			want: "one",
		},
		{
			args: args{
				header:     header,
				headerName: "two",
			},
			want: "one two",
		},
		{
			args: args{
				header:     header,
				headerName: "three",
			},
			want: "one two three",
		},
	}
	for _, tt := range tests {
		t.Run(tt.args.headerName, func(t *testing.T) {
			got, err := getHeaderValue(tt.args.header, tt.args.headerName)
			if err != nil {
				t.Errorf("getHeaderValue() error: %s", err)
			} else if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("getHeaderValue(): -want +got: %s", diff)
			}
		})
	}
}

func Test_getHeaderValue_error(t *testing.T) {
	header := map[string][]string{"one": {"one"}}
	tests := []struct {
		header     map[string][]string
		headerName string
	}{
		{
			header:     header,
			headerName: "bogusheadername",
		},
	}
	for _, tt := range tests {
		t.Run(tt.headerName, func(t *testing.T) {
			got, err := getHeaderValue(tt.header, tt.headerName)
			if err == nil {
				t.Errorf("getHeaderValue() did not return error when expected; got: %s", got)
			}
		})
	}
}

var (
	testBodyJSON       = json.RawMessage(`{"one": "onevalue", "two": {"two": "twovalue"}, "three": {"three": {"three": {"three": {"three": "threevalue"}}}}}`)
	paramNoBodyPathVar = pipelinev1.Param{
		Name:  "paramNoBodyPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: "bar"},
	}
	wantParamNoBodyPathVar = pipelinev1.Param{
		Name:  "paramNoBodyPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: "bar"},
	}
	paramOneBodyPathVar = pipelinev1.Param{
		Name:  "paramOneBodyPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-$(body.one)-bar"},
	}
	wantParamOneBodyPathVar = pipelinev1.Param{
		Name:  "paramOneBodyPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-onevalue-bar"},
	}
	paramMultipleIdenticalBodyPathVars = pipelinev1.Param{
		Name:  "paramMultipleIdenticalBodyPathVars",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-$(body.one)-$(body.one)-$(body.one)-bar"},
	}
	wantParamMultipleIdenticalBodyPathVars = pipelinev1.Param{
		Name:  "paramMultipleIdenticalBodyPathVars",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-onevalue-onevalue-onevalue-bar"},
	}
	paramMultipleUniqueBodyPathVars = pipelinev1.Param{
		Name:  "paramMultipleUniqueBodyPathVars",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-$(body.one)-$(body.two.two)-$(body.three.three.three.three.three)-bar"},
	}
	wantParamMultipleUniqueBodyPathVars = pipelinev1.Param{
		Name:  "paramMultipleUniqueBodyPathVars",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-onevalue-twovalue-threevalue-bar"},
	}
	paramSubobjectBodyPathVar = pipelinev1.Param{
		Name:  "paramSubobjectBodyPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-$(body.three)-bar"},
	}
	wantParamSubobjectBodyPathVar = pipelinev1.Param{
		Name:  "paramSubobjectBodyPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: `bar-{\"three\": {\"three\": {\"three\": {\"three\": \"threevalue\"}}}}-bar`},
	}
	paramEntireBodyPathVar = pipelinev1.Param{
		Name:  "paramEntireBodyPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-$(body)-bar"},
	}
	wantParamEntireBodyPathVar = pipelinev1.Param{
		Name:  "paramEntireBodyPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: `bar-{\"one\": \"onevalue\", \"two\": {\"two\": \"twovalue\"}, \"three\": {\"three\": {\"three\": {\"three\": {\"three\": \"threevalue\"}}}}}-bar`},
	}
	paramOneBogusBodyPathVar = pipelinev1.Param{
		Name:  "paramOneBogusBodyPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-$(body.bogus.path)-bar"},
	}
	paramMultipleBogusBodyPathVars = pipelinev1.Param{
		Name:  "paramMultipleBogusBodyPathVars",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-$(body.bogus.path)-$(body.two.bogus)-$(body.three.bogus)-bar"},
	}
)

func Test_applyBodyToParam(t *testing.T) {
	type args struct {
		body  []byte
		param pipelinev1.Param
	}
	tests := []struct {
		args args
		want pipelinev1.Param
	}{
		{
			args: args{body: []byte{}, param: paramNoBodyPathVar},
			want: wantParamNoBodyPathVar,
		},
		{
			args: args{body: testBodyJSON, param: paramOneBodyPathVar},
			want: wantParamOneBodyPathVar,
		},
		{
			args: args{body: testBodyJSON, param: paramMultipleIdenticalBodyPathVars},
			want: wantParamMultipleIdenticalBodyPathVars,
		},
		{
			args: args{body: testBodyJSON, param: paramMultipleUniqueBodyPathVars},
			want: wantParamMultipleUniqueBodyPathVars,
		},
		{
			args: args{body: testBodyJSON, param: paramEntireBodyPathVar},
			want: wantParamEntireBodyPathVar,
		},
		{
			args: args{body: testBodyJSON, param: paramSubobjectBodyPathVar},
			want: wantParamSubobjectBodyPathVar,
		},
	}
	for _, tt := range tests {
		t.Run(tt.args.param.Value.StringVal, func(t *testing.T) {
			got, err := applyBodyToParam(tt.args.body, tt.args.param)
			if err != nil {
				t.Errorf("applyBodyToParam() error = %v", err)
			} else if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("applyBodyToParam(): -want +got: %s", diff)
			}
		})
	}
}

func Test_applyBodyToParam_error(t *testing.T) {
	tests := []struct {
		body  []byte
		param pipelinev1.Param
	}{
		{
			body:  testBodyJSON,
			param: paramOneBogusBodyPathVar,
		},
		{
			body:  testBodyJSON,
			param: paramMultipleBogusBodyPathVars,
		},
	}
	for _, tt := range tests {
		t.Run(tt.param.Value.StringVal, func(t *testing.T) {
			got, err := applyBodyToParam(tt.body, tt.param)
			if err == nil {
				t.Errorf("applyBodyToParam() did not return error when expected; got: %v", got)
			}
		})
	}
}

func Test_ApplyBodyToParams(t *testing.T) {
	type args struct {
		body   []byte
		params []pipelinev1.Param
	}
	tests := []struct {
		name string
		args args
		want []pipelinev1.Param
	}{
		{
			name: "empty params",
			args: args{
				body:   testBodyJSON,
				params: []pipelinev1.Param{},
			},
			want: []pipelinev1.Param{},
		},
		{
			name: "one param",
			args: args{
				body:   testBodyJSON,
				params: []pipelinev1.Param{paramOneBodyPathVar},
			},
			want: []pipelinev1.Param{wantParamOneBodyPathVar},
		},
		{
			name: "multiple params",
			args: args{
				body: testBodyJSON,
				params: []pipelinev1.Param{
					paramOneBodyPathVar,
					paramMultipleUniqueBodyPathVars,
					paramSubobjectBodyPathVar,
				},
			},
			want: []pipelinev1.Param{
				wantParamOneBodyPathVar,
				wantParamMultipleUniqueBodyPathVars,
				wantParamSubobjectBodyPathVar,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyBodyToParams(tt.args.body, tt.args.params)
			if err != nil {
				t.Errorf("ApplyBodyToParams() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ApplyBodyToParams(): -want +got: %s", diff)
			}
		})
	}
}

func Test_ApplyBodyToParams_error(t *testing.T) {
	type args struct {
		body   []byte
		params []pipelinev1.Param
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "error one bodypath not found",
			args: args{
				body: testBodyJSON,
				params: []pipelinev1.Param{
					paramOneBogusBodyPathVar,
					paramMultipleUniqueBodyPathVars,
					paramSubobjectBodyPathVar,
				},
			},
		},
		{
			name: "error multiple bodypaths not found",
			args: args{
				body: testBodyJSON,
				params: []pipelinev1.Param{
					paramOneBogusBodyPathVar,
					paramMultipleBogusBodyPathVars,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyBodyToParams(tt.args.body, tt.args.params)
			if err == nil {
				t.Errorf("ApplyBodyToParams() did not return error when expected; got: %v", got)
			}
		})
	}
}

func Test_applyHeaderToParams(t *testing.T) {
	header := map[string][]string{"one": {"one"}, "two": {"one", "two"}, "three": {"one", "two", "three"}}
	type args struct {
		header map[string][]string
		param  pipelinev1.Param
	}
	tests := []struct {
		name string
		args args
		want pipelinev1.Param
	}{
		{
			name: "empty",
			args: args{
				header: header,
				param:  pipelinev1.Param{},
			},
			want: pipelinev1.Param{},
		},
		{
			name: "no header vars",
			args: args{
				header: header,
				param: pipelinev1.Param{
					Name:  "noHeaderVars",
					Value: pipelinev1.ArrayOrString{StringVal: "foo"},
				},
			},
			want: pipelinev1.Param{
				Name:  "noHeaderVars",
				Value: pipelinev1.ArrayOrString{StringVal: "foo"},
			},
		},
		{
			name: "one header var",
			args: args{
				header: header,
				param: pipelinev1.Param{
					Name:  "oneHeaderVar",
					Value: pipelinev1.ArrayOrString{StringVal: "$(header.one)"},
				},
			},
			want: pipelinev1.Param{
				Name:  "oneHeaderVar",
				Value: pipelinev1.ArrayOrString{StringVal: "one"},
			},
		},
		{
			name: "multiple header vars",
			args: args{
				header: header,
				param: pipelinev1.Param{
					Name:  "multipleHeaderVars",
					Value: pipelinev1.ArrayOrString{StringVal: "$(header.one)-$(header.two)-$(header.three)"},
				},
			},
			want: pipelinev1.Param{
				Name:  "multipleHeaderVars",
				Value: pipelinev1.ArrayOrString{StringVal: `one-one two-one two three`},
			},
		},
		{
			name: "identical header vars",
			args: args{
				header: header,
				param: pipelinev1.Param{
					Name:  "identicalHeaderVars",
					Value: pipelinev1.ArrayOrString{StringVal: "$(header.one)-$(header.one)-$(header.one)"},
				},
			},
			want: pipelinev1.Param{
				Name:  "identicalHeaderVars",
				Value: pipelinev1.ArrayOrString{StringVal: `one-one-one`},
			},
		},
		{
			name: "entire header var",
			args: args{
				header: header,
				param: pipelinev1.Param{
					Name:  "entireHeaderVar",
					Value: pipelinev1.ArrayOrString{StringVal: "$(header)"},
				},
			},
			want: pipelinev1.Param{
				Name:  "entireHeaderVar",
				Value: pipelinev1.ArrayOrString{StringVal: `{\"one\":[\"one\"],\"three\":[\"one\",\"two\",\"three\"],\"two\":[\"one\",\"two\"]}`},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyHeaderToParam(tt.args.header, tt.args.param)
			if err != nil {
				t.Errorf("applyHeaderToParam() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("applyHeaderToParam(): -want +got: %s", diff)
			}
		})
	}
}

func Test_applyHeaderToParams_error(t *testing.T) {
	header := map[string][]string{"one": {"one"}}
	type args struct {
		header map[string][]string
		param  pipelinev1.Param
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "error header not found",
			args: args{
				header: header,
				param: pipelinev1.Param{
					Name:  "oneBogusHeader",
					Value: pipelinev1.ArrayOrString{StringVal: "$(header.bogus)"},
				},
			},
		},
		{
			name: "error multiple headers not found",
			args: args{
				header: header,
				param: pipelinev1.Param{
					Name:  "multipleBogusHeaders",
					Value: pipelinev1.ArrayOrString{StringVal: "$(header.one)-$(header.bogus1)-$(header.bogus2)-$(header.bogus3)"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyHeaderToParam(tt.args.header, tt.args.param)
			if err == nil {
				t.Errorf("applyHeaderToParam() did not return error when expected; got: %v", got)
			}
		})
	}
}

func Test_NewResources(t *testing.T) {
	type args struct {
		body    []byte
		header  map[string][]string
		binding ResolvedBinding
	}
	tests := []struct {
		name string
		args args
		want []json.RawMessage
	}{
		{
			name: "empty",
			args: args{
				body:   json.RawMessage{},
				header: map[string][]string{},
				binding: ResolvedBinding{
					TriggerTemplate: bldr.TriggerTemplate("tt", "namespace"),
					TriggerBindings: []*triggersv1.TriggerBinding{bldr.TriggerBinding("tb", "namespace")},
				},
			},
			want: []json.RawMessage{},
		},
		{
			name: "one resource template",
			args: args{
				body:   json.RawMessage(`{"foo": "bar"}`),
				header: map[string][]string{"one": {"1"}},
				binding: ResolvedBinding{
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
		},
		{
			name: "multiple resource templates",
			args: args{
				body:   json.RawMessage(`{"foo": "bar"}`),
				header: map[string][]string{"one": {"1"}},
				binding: ResolvedBinding{
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
		},
		{
			name: "one resource template with one uid",
			args: args{
				body: json.RawMessage(`{"foo": "bar"}`),
				binding: ResolvedBinding{
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
		},
		{
			name: "one resource template with three uid",
			args: args{
				body: json.RawMessage(`{"foo": "bar"}`),
				binding: ResolvedBinding{
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
		},
		{
			name: "multiple resource templates with multiple uid",
			args: args{
				body: json.RawMessage(`{"foo": "bar"}`),
				binding: ResolvedBinding{
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
				json.RawMessage(`{"rt2": "default2-cbhtc"}`),
				json.RawMessage(`{"rt3": "rt3"}`),
			},
		},
		{
			name: "one resource template multiple bindings",
			args: args{
				body:   json.RawMessage(`{"foo": "bar"}`),
				header: map[string][]string{"one": {"1"}},
				binding: ResolvedBinding{
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This seeds Uid() to return 'cbhtc'
			rand.Seed(0)
			got, err := NewResources(tt.args.body, tt.args.header, tt.args.binding)
			if err != nil {
				t.Errorf("NewResources() returned unexpected error: %s", err)
			} else if diff := cmp.Diff(tt.want, got); diff != "" {
				stringDiff := cmp.Diff(convertJSONRawMessagesToString(tt.want), convertJSONRawMessagesToString(got))
				t.Errorf("NewResources(): -want +got: %s", stringDiff)
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
		binding  ResolvedBinding
	}{
		{
			name: "bodypath not found in body",
			body: json.RawMessage(`{"foo": "bar"}`),
			binding: ResolvedBinding{
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
		},
		{
			name:   "header not found in event",
			body:   json.RawMessage(`{"foo": "bar"}`),
			header: map[string][]string{"One": {"one"}},
			binding: ResolvedBinding{
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
		},
		{
			name: "merge params error",
			elParams: []pipelinev1.Param{
				{
					Name:  "param1",
					Value: pipelinev1.ArrayOrString{StringVal: "value1", Type: pipelinev1.ParamTypeString},
				},
			},
			binding: ResolvedBinding{
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
		},
		{
			name: "conflicting bindings",
			binding: ResolvedBinding{
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
			got, err := NewResources(tt.body, tt.header, tt.binding)
			if err == nil {
				t.Errorf("NewResources() did not return error when expected; got: %s", got)
			}
		})
	}
}

func TestExamplesForBodyPathVariables(t *testing.T) {
	var testNames []string
	var payloads [][]byte
	// Populates payloads using examples
	err := filepath.Walk("./examples", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.Errorf("Failure accessing path %q: %v\n", path, err)
			return err
		}
		if !info.IsDir() {
			t.Logf("Reading %s", path)
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			var buffer bytes.Buffer
			err = json.Compact(&buffer, b)
			if err != nil {
				t.Errorf("Error making json compact: %s", err)
			}
			payloads = append(payloads, buffer.Bytes())
			fileNameTrimmed := strings.TrimSuffix(path, ".json")
			testNames = append(testNames, strings.Replace(strings.Title(fileNameTrimmed), "/", "", -1))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Unable to load example payloads: %s", err)
	}

	// Validate all fields can be pulled for each respective payload
	for i := range payloads {
		iCopy := i
		t.Run(testNames[i], func(t *testing.T) {
			// Grab bindings and expected values
			bodyPaths, expectedValues, err := bodyPathDigger(payloads[iCopy], "")
			if err != nil {
				t.Errorf("Failed to generate bindings for %s payload", testNames[iCopy])
				return
			}

			// Grab actual values
			for i := range bodyPaths {
				gotValue, err := getBodyPathValue(payloads[iCopy], bodyPaths[i])
				if err != nil {
					t.Errorf("Error getting value for bodyPath %s: %s", bodyPaths[i], err)
					continue
				}
				// Format so compare does not care about order of keys
				gotBytes := []byte(strings.Replace(gotValue, `\"`, `"`, -1))
				wantBytes := []byte(strings.Replace(expectedValues[i], `\"`, `"`, -1))
				if json.Valid(gotBytes) && json.Valid(wantBytes) {
					// Make all formatting compact to compare content, not formatting
					var wantUnmarshalled interface{}
					if err = json.Unmarshal([]byte(wantBytes), &wantUnmarshalled); err != nil {
						t.Errorf("Error unmarshalling wantValue %s: %s", wantBytes, err)
						continue
					}
					want, _ := json.Marshal(wantUnmarshalled)
					var gotUnmarshalled interface{}
					if err = json.Unmarshal(gotBytes, &gotUnmarshalled); err != nil {
						t.Errorf("Error unmarshalling gotValue %s: %s", gotBytes, err)
						continue
					}
					got, _ := json.Marshal(gotUnmarshalled)
					if diff := cmp.Diff(string(want), string(got)); diff != "" {
						t.Errorf("Error for bodyPath %s: diff -want +got: %s", bodyPaths[i], diff)
					}
				} else {
					if diff := cmp.Diff(expectedValues[i], gotValue); diff != "" {
						t.Errorf("Error for bodyPath %s -want +got: %s", bodyPaths[i], diff)
					}
				}
			}
		})
	}
}

// Test_bodyPathDigger tests bodyPathDigger to ensure all possible body paths are returned
// The simple digger example payload is space compacted
func Test_bodyPathDigger(t *testing.T) {
	// Small example file/payload used to validate the functionality of bodyPathDigger
	filePath := "examples/digger.json"
	t.Logf("Reading %s", filePath)
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Error reading file %s: %s", filePath, err)
	}
	// Create assertion map
	// The example digger payload fields have been predefined as space compacted
	pathValueMap := map[string]string{
		"":    strings.Replace(string(b), `"`, `\"`, -1),
		"a":   "a",
		"b":   "true",
		"c":   "30",
		"d":   "[]",
		"e":   `[\"a\"]`,
		"f":   "[false]",
		"g":   "[3]",
		"h":   `[{\"a\":\"a\"}]`,
		"i":   `{\"a\":\"a\"}`,
		"i.a": "a",
		"j":   "",
	}
	// Grab values
	bodyPaths, actualValues, err := bodyPathDigger(b, "")
	if err != nil {
		t.Fatalf("Failed to generate body paths for %s payload", filePath)
	}

	// Ensure sizing is the same
	if len(bodyPaths) != len(pathValueMap) {
		t.Fatalf("Length of bodyPaths[%d] did not match expected[%d]", len(bodyPaths), len(pathValueMap))
	}
	if len(actualValues) != len(pathValueMap) {
		t.Fatalf("Length of actualValues[%d] did not match expected[%d]", len(actualValues), len(pathValueMap))
	}
	// Validate against assertion map
	for i := 0; i < len(bodyPaths); i++ {
		expectedValue := pathValueMap[bodyPaths[i]]
		actualValue := actualValues[i]
		if diff := cmp.Diff(expectedValue, actualValue); diff != "" {
			t.Errorf("Diff: -want +got: %s", diff)
		}
	}
}

// bodyPathDigger returns all possible body paths and corresponding expected values from the payload
// Digs into map recursively whenever the value is a json object
// Inherent to the marshalling of json, expectedValues cannot by guaranteed in the same order as payload
func bodyPathDigger(payload []byte, location string) (bodyPaths []string, expectedValues []string, err error) {
	// Trim quotes if they exist ("value" -> value)
	value := strings.TrimPrefix(strings.TrimSuffix(string(payload), "\""), "\"")
	// Escape double quotes
	value = strings.Replace(value, `"`, `\"`, -1)
	// Add the entire event payload/base
	bodyPaths = append(bodyPaths, location)
	expectedValues = append(expectedValues, value)

	// Store body as map to make it iterable
	m := make(map[string]interface{})
	err = json.Unmarshal(payload, &m)
	if err != nil {
		// Payload is a value, so stop recursion
		return bodyPaths, expectedValues, nil
	}
	// Iterate over fields (potentially recursively) to capture all body paths and expected values
	for field, value := range m {
		var currentLocation string
		if location == "" {
			currentLocation = field
		} else {
			currentLocation = fmt.Sprintf("%s.%s", location, field)
		}
		b, err := json.Marshal(value)
		if err != nil {
			return nil, nil, xerrors.Errorf("Failed to marshal value %v: %s", value, err)
		}
		nestedPaths, nestedValues, err := bodyPathDigger(b, currentLocation)
		if err != nil {
			return nil, nil, err
		}
		bodyPaths, expectedValues = append(bodyPaths, nestedPaths...), append(expectedValues, nestedValues...)
	}
	return bodyPaths, expectedValues, nil
}
