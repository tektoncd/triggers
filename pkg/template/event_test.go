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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	bldr "github.com/tektoncd/triggers/test/builder"
	"golang.org/x/xerrors"

	"k8s.io/apimachinery/pkg/util/rand"
)

func Test_EventPathVarRegex(t *testing.T) {
	tests := []string{
		"$(event)",
		"$(event.a-b)",
		"$(event.a1)",
		"$(event.a.b)",
		"$(event.a.b.c)",
	}
	for _, eventPathVar := range tests {
		t.Run(eventPathVar, func(t *testing.T) {
			if !eventPathVarRegex.MatchString(eventPathVar) {
				t.Errorf("eventPathVarRegex.MatchString(%s) = false, want = true", eventPathVar)
			}
		})
	}
}

func Test_EventPathVarRegex_invalid(t *testing.T) {
	tests := []string{
		"$event",
		"$[event]",
		"${event}",
		"$(event.)",
		"$(event..)",
		"$(event.$a)",
		"event.a",
		"event",
		"${{event}",
		"${event",
	}
	for _, eventPathVar := range tests {
		t.Run(eventPathVar, func(t *testing.T) {
			if eventPathVarRegex.MatchString(eventPathVar) {
				t.Errorf("eventPathVarRegex.MatchString(%s) = true, want = false", eventPathVar)
			}
		})
	}
}

func Test_GetEventPathFromVar(t *testing.T) {
	tests := []struct {
		eventPathVar string
		want         string
	}{
		{eventPathVar: "$(event)", want: ""},
		{eventPathVar: "$(event.a-b)", want: "a-b"},
		{eventPathVar: "$(event.a1)", want: "a1"},
		{eventPathVar: "$(event.a.b)", want: "a.b"},
		{eventPathVar: "$(event.a.b.c)", want: "a.b.c"},
	}
	for _, tt := range tests {
		t.Run(tt.eventPathVar, func(t *testing.T) {
			if eventPath := getEventPathFromVar(tt.eventPathVar); eventPath != tt.want {
				t.Errorf("getEventPathFromVar() = %s, want = %s", eventPath, tt.want)
			}
		})
	}
}

func Test_getEventPathValue(t *testing.T) {
	event := `{"empty": "", "null": null, "one": "one", "two": {"two": "twovalue"}, "three": {"three": {"three": {"three": {"three": "threevalue"}}}}}`
	eventJSON := json.RawMessage(event)
	type args struct {
		event     []byte
		eventPath string
	}
	tests := []struct {
		args args
		want string
	}{
		{
			args: args{
				event:     eventJSON,
				eventPath: "",
			},
			want: event,
		},
		{
			args: args{
				event:     eventJSON,
				eventPath: "one",
			},
			want: "one",
		},
		{
			args: args{
				event:     eventJSON,
				eventPath: "two",
			},
			want: `{"two": "twovalue"}`,
		},
		{
			args: args{
				event:     eventJSON,
				eventPath: "three.three.three.three.three",
			},
			want: "threevalue",
		},
		{
			args: args{
				event:     eventJSON,
				eventPath: "empty",
			},
			want: "",
		},
		{
			args: args{
				event:     eventJSON,
				eventPath: "null",
			},
			want: "null",
		},
	}
	for _, tt := range tests {
		t.Run(tt.args.eventPath, func(t *testing.T) {
			got, err := getEventPathValue(tt.args.event, tt.args.eventPath)
			if err != nil {
				t.Errorf("getEventPathValue() error: %s", err)
			} else if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("getEventPathValue(): -want +got: %s", diff)
			}
		})
	}
}

func Test_getEventPathValue_error(t *testing.T) {
	eventJSON := json.RawMessage(`{"one": "onevalue", "two": {"two": "twovalue"}, "three": {"three": {"three": {"three": {"three": "threevalue"}}}}}`)
	tests := []struct {
		event     []byte
		eventPath string
	}{
		{
			event:     eventJSON,
			eventPath: "boguspath",
		},
		{
			event:     eventJSON,
			eventPath: "two.bogus",
		},
		{
			event:     eventJSON,
			eventPath: "three.three.bogus.three",
		},
	}
	for _, tt := range tests {
		t.Run(tt.eventPath, func(t *testing.T) {
			got, err := getEventPathValue(tt.event, tt.eventPath)
			if err == nil {
				t.Errorf("getEventPathValue() did not return error when expected; got: %s", got)
			}
		})
	}
}

var (
	testEventJSON       = json.RawMessage(`{"one": "onevalue", "two": {"two": "twovalue"}, "three": {"three": {"three": {"three": {"three": "threevalue"}}}}}`)
	paramNoEventPathVar = pipelinev1.Param{
		Name:  "paramNoEventPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: "bar"},
	}
	wantParamNoEventPathVar = pipelinev1.Param{
		Name:  "paramNoEventPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: "bar"},
	}
	paramOneEventPathVar = pipelinev1.Param{
		Name:  "paramOneEventPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-$(event.one)-bar"},
	}
	wantParamOneEventPathVar = pipelinev1.Param{
		Name:  "paramOneEventPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-onevalue-bar"},
	}
	paramMultipleIdenticalEventPathVars = pipelinev1.Param{
		Name:  "paramMultipleIdenticalEventPathVars",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-$(event.one)-$(event.one)-$(event.one)-bar"},
	}
	wantParamMultipleIdenticalEventPathVars = pipelinev1.Param{
		Name:  "paramMultipleIdenticalEventPathVars",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-onevalue-onevalue-onevalue-bar"},
	}
	paramMultipleUniqueEventPathVars = pipelinev1.Param{
		Name:  "paramMultipleUniqueEventPathVars",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-$(event.one)-$(event.two.two)-$(event.three.three.three.three.three)-bar"},
	}
	wantParamMultipleUniqueEventPathVars = pipelinev1.Param{
		Name:  "paramMultipleUniqueEventPathVars",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-onevalue-twovalue-threevalue-bar"},
	}
	paramSubobjectEventPathVar = pipelinev1.Param{
		Name:  "paramSubobjectEventPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-$(event.three)-bar"},
	}
	wantParamSubobjectEventPathVar = pipelinev1.Param{
		Name:  "paramSubobjectEventPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: `bar-{"three": {"three": {"three": {"three": "threevalue"}}}}-bar`},
	}
	paramEntireEventEventPathVar = pipelinev1.Param{
		Name:  "paramEntireEventEventPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-$(event)-bar"},
	}
	wantParamEntireEventEventPathVar = pipelinev1.Param{
		Name:  "paramEntireEventEventPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: `bar-{"one": "onevalue", "two": {"two": "twovalue"}, "three": {"three": {"three": {"three": {"three": "threevalue"}}}}}-bar`},
	}
	paramOneBogusEventPathVar = pipelinev1.Param{
		Name:  "paramOneBogusEventPathVar",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-$(event.bogus.path)-bar"},
	}
	paramMultipleBogusEventPathVars = pipelinev1.Param{
		Name:  "paramMultipleBogusEventPathVars",
		Value: pipelinev1.ArrayOrString{StringVal: "bar-$(event.bogus.path)-$(event.two.bogus)-$(event.three.bogus)-bar"},
	}
)

func Test_applyEventToParam(t *testing.T) {
	type args struct {
		event []byte
		param pipelinev1.Param
	}
	tests := []struct {
		args args
		want pipelinev1.Param
	}{
		{
			args: args{event: []byte{}, param: paramNoEventPathVar},
			want: wantParamNoEventPathVar,
		},
		{
			args: args{event: testEventJSON, param: paramOneEventPathVar},
			want: wantParamOneEventPathVar,
		},
		{
			args: args{event: testEventJSON, param: paramMultipleIdenticalEventPathVars},
			want: wantParamMultipleIdenticalEventPathVars,
		},
		{
			args: args{event: testEventJSON, param: paramMultipleUniqueEventPathVars},
			want: wantParamMultipleUniqueEventPathVars,
		},
		{
			args: args{event: testEventJSON, param: paramEntireEventEventPathVar},
			want: wantParamEntireEventEventPathVar,
		},
		{
			args: args{event: testEventJSON, param: paramSubobjectEventPathVar},
			want: wantParamSubobjectEventPathVar,
		},
	}
	for _, tt := range tests {
		t.Run(tt.args.param.Value.StringVal, func(t *testing.T) {
			got, err := applyEventToParam(tt.args.event, tt.args.param)
			if err != nil {
				t.Errorf("applyEventToParam() error = %v", err)
			} else if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("applyEventToParam(): -want +got: %s", diff)
			}
		})
	}
}

func Test_applyEventToParam_error(t *testing.T) {
	tests := []struct {
		event []byte
		param pipelinev1.Param
	}{
		{
			event: testEventJSON,
			param: paramOneBogusEventPathVar,
		},
		{
			event: testEventJSON,
			param: paramMultipleBogusEventPathVars,
		},
	}
	for _, tt := range tests {
		t.Run(tt.param.Value.StringVal, func(t *testing.T) {
			got, err := applyEventToParam(tt.event, tt.param)
			if err == nil {
				t.Errorf("applyEventToParam() did not return error when expected; got: %v", got)
			}
		})
	}
}

func Test_ApplyEventToParams(t *testing.T) {
	type args struct {
		event  []byte
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
				event:  testEventJSON,
				params: []pipelinev1.Param{},
			},
			want: []pipelinev1.Param{},
		},
		{
			name: "one param",
			args: args{
				event:  testEventJSON,
				params: []pipelinev1.Param{paramOneEventPathVar},
			},
			want: []pipelinev1.Param{wantParamOneEventPathVar},
		},
		{
			name: "multiple params",
			args: args{
				event: testEventJSON,
				params: []pipelinev1.Param{
					paramOneEventPathVar,
					paramMultipleUniqueEventPathVars,
					paramSubobjectEventPathVar,
				},
			},
			want: []pipelinev1.Param{
				wantParamOneEventPathVar,
				wantParamMultipleUniqueEventPathVars,
				wantParamSubobjectEventPathVar,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyEventToParams(tt.args.event, tt.args.params)
			if err != nil {
				t.Errorf("ApplyEventToParams() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ApplyEventToParams(): -want +got: %s", diff)
			}
		})
	}
}

func Test_ApplyEventToParams_error(t *testing.T) {
	type args struct {
		event  []byte
		params []pipelinev1.Param
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "error one eventpath not found",
			args: args{
				event: testEventJSON,
				params: []pipelinev1.Param{
					paramOneBogusEventPathVar,
					paramMultipleUniqueEventPathVars,
					paramSubobjectEventPathVar,
				},
			},
		},
		{
			name: "error multiple eventpaths not found",
			args: args{
				event: testEventJSON,
				params: []pipelinev1.Param{
					paramOneBogusEventPathVar,
					paramMultipleBogusEventPathVars,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyEventToParams(tt.args.event, tt.args.params)
			if err == nil {
				t.Errorf("ApplyEventToParams() did not return error when expected; got: %v", got)
			}
		})
	}
}

func Test_NewResources(t *testing.T) {
	tests := []struct {
		name    string
		event   []byte
		binding ResolvedBinding
		want    []json.RawMessage
	}{
		{
			name:  "empty",
			event: json.RawMessage{},
			binding: ResolvedBinding{
				TriggerTemplate: bldr.TriggerTemplate("tt", "namespace"),
				TriggerBinding:  bldr.TriggerBinding("tb", "namespace"),
			},
			want: []json.RawMessage{},
		},
		{
			name:  "one resource template",
			event: json.RawMessage(`{"foo": "bar"}`),
			binding: ResolvedBinding{
				TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
					bldr.TriggerTemplateSpec(
						bldr.TriggerTemplateParam("param1", "description", ""),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt1": "$(params.param1)"}`)),
					),
				),
				TriggerBinding: bldr.TriggerBinding("tb", "namespace",
					bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("param1", "$(event.foo)"),
					),
				),
			},
			want: []json.RawMessage{
				json.RawMessage(`{"rt1": "bar"}`),
			},
		},
		{
			name:  "multiple resource templates",
			event: json.RawMessage(`{"foo": "bar"}`),
			binding: ResolvedBinding{
				TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
					bldr.TriggerTemplateSpec(
						bldr.TriggerTemplateParam("param1", "description", ""),
						bldr.TriggerTemplateParam("param2", "description", "default2"),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt1": "$(params.param1)"}`)),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt2": "$(params.param2)"}`)),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt3": "rt3"}`)),
					),
				),
				TriggerBinding: bldr.TriggerBinding("tb", "namespace",
					bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("param1", "$(event.foo)"),
					),
				),
			},
			want: []json.RawMessage{
				json.RawMessage(`{"rt1": "bar"}`),
				json.RawMessage(`{"rt2": "default2"}`),
				json.RawMessage(`{"rt3": "rt3"}`),
			},
		},
		{
			name:  "one resource template with one uid",
			event: json.RawMessage(`{"foo": "bar"}`),
			binding: ResolvedBinding{
				TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
					bldr.TriggerTemplateSpec(
						bldr.TriggerTemplateParam("param1", "description", ""),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt1": "$(params.param1)-$(uid)"}`)),
					),
				),
				TriggerBinding: bldr.TriggerBinding("tb", "namespace",
					bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("param1", "$(event.foo)"),
					),
				),
			},
			want: []json.RawMessage{
				json.RawMessage(`{"rt1": "bar-cbhtc"}`),
			},
		},
		{
			name:  "one resource template with three uid",
			event: json.RawMessage(`{"foo": "bar"}`),
			binding: ResolvedBinding{
				TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
					bldr.TriggerTemplateSpec(
						bldr.TriggerTemplateParam("param1", "description", ""),
						bldr.TriggerResourceTemplate(json.RawMessage(`{"rt1": "$(params.param1)-$(uid)-$(uid)", "rt2": "$(uid)"}`)),
					),
				),
				TriggerBinding: bldr.TriggerBinding("tb", "namespace",
					bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("param1", "$(event.foo)"),
					),
				),
			},
			want: []json.RawMessage{
				json.RawMessage(`{"rt1": "bar-cbhtc-cbhtc", "rt2": "cbhtc"}`),
			},
		},
		{
			name:  "multiple resource templates with multiple uid",
			event: json.RawMessage(`{"foo": "bar"}`),
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
				TriggerBinding: bldr.TriggerBinding("tb", "namespace",
					bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("param1", "$(event.foo)"),
					),
				),
			},
			want: []json.RawMessage{
				json.RawMessage(`{"rt1": "bar-cbhtc", "cbhtc": "cbhtc"}`),
				json.RawMessage(`{"rt2": "default2-cbhtc"}`),
				json.RawMessage(`{"rt3": "rt3"}`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This seeds Uid() to return 'cbhtc'
			rand.Seed(0)
			got, err := NewResources(tt.event, tt.binding)
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
		name    string
		event   []byte
		binding ResolvedBinding
	}{
		{
			name:  "eventpath not found in event",
			event: json.RawMessage(`{"foo": "bar"}`),
			binding: ResolvedBinding{
				TriggerTemplate: bldr.TriggerTemplate("tt", "namespace",
					bldr.TriggerTemplateSpec(
						bldr.TriggerTemplateParam("param1", "description", ""),
					),
				),
				TriggerBinding: bldr.TriggerBinding("tb", "namespace",
					bldr.TriggerBindingSpec(
						bldr.TriggerBindingParam("param1", "$(event.bogusvalue)"),
					),
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewResources(tt.event, tt.binding)
			if err == nil {
				t.Errorf("NewResources() did not return error when expected; got: %s", got)
			}
		})
	}
}

func TestExamplesForEventPathVariables(t *testing.T) {
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
			payloads = append(payloads, b)
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
			eventPaths, expectedValues, err := eventPathDigger(payloads[iCopy], "")
			if err != nil {
				t.Errorf("Failed to generate event bindings for %s payload", testNames[iCopy])
				return
			}

			// Grab actual values
			for i := range eventPaths {
				gotValue, err := getEventPathValue(payloads[iCopy], eventPaths[i])
				if err != nil {
					t.Errorf("Error getting value for eventPath %s: %s", eventPaths[i], err)
					continue
				}
				if json.Valid([]byte(gotValue)) && json.Valid([]byte(expectedValues[i])) {
					// Make all formatting compact to compare content, not formatting
					var wantUnmarshalled interface{}
					if err = json.Unmarshal([]byte(expectedValues[i]), &wantUnmarshalled); err != nil {
						t.Errorf("Error unmarshalling wantValue %s: %s", expectedValues[i], err)
						continue
					}
					want, _ := json.Marshal(wantUnmarshalled)
					var gotUnmarshalled interface{}
					if err = json.Unmarshal([]byte(gotValue), &gotUnmarshalled); err != nil {
						t.Errorf("Error unmarshalling gotValue %s: %s", gotValue, err)
						continue
					}
					got, _ := json.Marshal(gotUnmarshalled)
					if diff := cmp.Diff(string(want), string(got)); diff != "" {
						t.Errorf("Error for eventPath %s: diff -want +got: %s", eventPaths[i], diff)
					}
				} else {
					if diff := cmp.Diff(expectedValues[i], gotValue); diff != "" {
						t.Errorf("Error for eventPath %s -want +got: %s", eventPaths[i], diff)
					}
				}
			}
		})
	}
}

// Test_eventPathDigger tests eventPathDigger to ensure all possible event paths are returned
// The simple digger example payload is space compacted
func Test_eventPathDigger(t *testing.T) {
	// Small example file/payload used to validate the functionality of eventBindingDigger
	filePath := "examples/digger.json"
	t.Logf("Reading %s", filePath)
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Error reading file %s: %s", filePath, err)
	}
	// Create assertion map
	// The example digger payload fields have been predefined as space compacted
	pathValueMap := map[string]string{
		"":    string(b),
		"a":   "a",
		"b":   "true",
		"c":   "30",
		"d":   "[]",
		"e":   `["a"]`,
		"f":   "[false]",
		"g":   "[3]",
		"h":   `[{"a":"a"}]`,
		"i":   `{"a":"a"}`,
		"i.a": "a",
		"j":   "",
	}
	// Grab values
	eventPaths, actualValues, err := eventPathDigger(b, "")
	if err != nil {
		t.Fatalf("Failed to generate event bindings for %s payload", filePath)
	}

	// Ensure sizing is the same
	if len(eventPaths) != len(pathValueMap) {
		t.Fatalf("Length of eventBindings[%d] did not match expected[%d]", len(eventPaths), len(pathValueMap))
	}
	if len(actualValues) != len(pathValueMap) {
		t.Fatalf("Length of actualValues[%d] did not match expected[%d]", len(actualValues), len(pathValueMap))
	}
	// Validate against assertion map
	for i := 0; i < len(eventPaths); i++ {
		expectedValue := pathValueMap[eventPaths[i]]
		actualValue := actualValues[i]
		if diff := cmp.Diff(expectedValue, actualValue); diff != "" {
			t.Errorf("Diff: -want +got: %s", diff)
		}
	}
}

// eventPathDigger returns all possible event paths and corresponding expected values from the payload
// Digs into map recursively whenever the value is a json object
// Inherent to the marshalling of json, expectedValues cannot by guaranteed in the same order as payload
func eventPathDigger(payload []byte, location string) (eventPaths []string, expectedValues []string, err error) {
	// Trim quotes if they exist ("value" -> value)
	value := strings.TrimPrefix(strings.TrimSuffix(string(payload), "\""), "\"")
	// Add the entire event payload/base
	eventPaths = append(eventPaths, location)
	expectedValues = append(expectedValues, value)

	// Store event as map to make it iterable
	m := make(map[string]interface{})
	err = json.Unmarshal(payload, &m)
	if err != nil {
		// Payload is a value, so stop recursion
		return eventPaths, expectedValues, nil
	}
	// Iterate over fields (potentially recursively) to capture all event bindings and expected values
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
		nestedPaths, nestedValues, err := eventPathDigger(b, currentLocation)
		if err != nil {
			return nil, nil, err
		}
		eventPaths, expectedValues = append(eventPaths, nestedPaths...), append(expectedValues, nestedValues...)
	}
	return eventPaths, expectedValues, nil
}
