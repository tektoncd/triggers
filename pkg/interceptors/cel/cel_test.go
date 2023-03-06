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

package cel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/go-cmp/cmp"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
)

const testNS = "testing-ns"

func TestInterceptor_Process(t *testing.T) {
	tests := []struct {
		name           string
		CEL            *triggersv1.CELInterceptor
		body           []byte
		extensions     map[string]interface{}
		wantExtensions map[string]interface{}
	}{{
		name: "simple body check with matching body",
		CEL: &triggersv1.CELInterceptor{
			Filter: "body.value == 'testing'",
		},
		body: json.RawMessage(`{"value":"testing"}`),
	}, {
		name: "simple header check with matching header",
		CEL: &triggersv1.CELInterceptor{
			Filter: "header['X-Test'][0] == 'test-value'",
		},
		body: json.RawMessage(`{}`),
	}, {
		name: "overloaded header check with case insensitive matching",
		CEL: &triggersv1.CELInterceptor{
			Filter: "header.match('x-test', 'test-value')",
		},
		body: json.RawMessage(`{}`),
	}, {
		name: "body and header check",
		CEL: &triggersv1.CELInterceptor{
			Filter: "header.match('x-test', 'test-value') && body.value == 'test'",
		},
		body: json.RawMessage(`{"value":"test"}`),
	}, {
		name: "body and header canonical check",
		CEL: &triggersv1.CELInterceptor{
			Filter: "header.canonical('x-test') == 'test-value' && body.value == 'test'",
		},
		body: json.RawMessage(`{"value":"test"}`),
	}, {
		name: "single overlay",
		CEL: &triggersv1.CELInterceptor{
			Filter: "body.value == 'test'",
			Overlays: []triggersv1.CELOverlay{
				{Key: "new", Expression: "body.value"},
			},
		},
		body: json.RawMessage(`{"value":"test"}`),
		wantExtensions: map[string]interface{}{
			"new": "test",
		},
	}, {
		name: "single overlay with no filter",
		CEL: &triggersv1.CELInterceptor{
			Overlays: []triggersv1.CELOverlay{
				{Key: "new", Expression: "body.ref.split('/')[2]"},
			},
		},
		body: json.RawMessage(`{"ref":"refs/head/master","name":"testing"}`),
		wantExtensions: map[string]interface{}{
			"new": "master",
		},
	}, {
		name: "overlay with string library functions",
		CEL: &triggersv1.CELInterceptor{
			Overlays: []triggersv1.CELOverlay{
				{Key: "new", Expression: "body.ref.split('/')[2]"},
				{Key: "replaced", Expression: "body.name.replace('ing','ed',0)"},
			},
		},
		body: json.RawMessage(`{"ref":"refs/head/master","name":"testing"}`),
		wantExtensions: map[string]interface{}{
			"new":      "master",
			"replaced": "testing",
		},
	}, {
		name: "multiple overlays",
		CEL: &triggersv1.CELInterceptor{
			Filter: "body.value == 'test'",
			Overlays: []triggersv1.CELOverlay{
				{Key: "test.one", Expression: "body.value"},
				{Key: "test.two", Expression: "body.value"},
			},
		},
		body: json.RawMessage(`{"value":"test"}`),
		// TODO: Fix extensions if key contains ., use sjson to merge
		wantExtensions: map[string]interface{}{
			"test": map[string]interface{}{
				"two": "test",
				"one": "test",
			},
		},
	}, {
		name: "nil body does not panic",
		CEL:  &triggersv1.CELInterceptor{Filter: "header.match('x-test', 'test-value')"},
		body: nil,
	}, {
		name: "incrementing an integer value",
		CEL: &triggersv1.CELInterceptor{
			Overlays: []triggersv1.CELOverlay{
				{Key: "val1", Expression: "body.count + 1.0"},
				{Key: "val2", Expression: "int(body.count) + 3"},
				{Key: "val3", Expression: "body.count + 3.5"},
				{Key: "val4", Expression: "body.measure * 3.0"},
			},
		},
		body: json.RawMessage(`{"count":1,"measure":1.7}`),
		wantExtensions: map[string]interface{}{
			"val4": 5.1,
			"val3": 4.5,
			"val2": float64(4),
			"val1": float64(2),
		},
	}, {
		name: "validating a secret",
		CEL: &triggersv1.CELInterceptor{
			Filter: "header.canonical('X-Secret-Token').compareSecret('token', 'test-secret', 'testing-ns')",
		},
		body: json.RawMessage(`{"count":1,"measure":1.7}`),
	}, {
		name: "validating a secret with a namespace and name",
		CEL: &triggersv1.CELInterceptor{
			Filter: "header.canonical('X-Secret-Token').compareSecret('token', 'test-secret', 'testing-ns') && body.count == 1.0",
		},
		body: json.RawMessage(`{"count":1,"measure":1.7}`),
	}, {
		name: "validating a secret in the default namespace",
		CEL: &triggersv1.CELInterceptor{
			Filter: "header.canonical('X-Secret-Token').compareSecret('token', 'test-secret') && body.count == 1.0",
		},
		body: json.RawMessage(`{"count":1,"measure":1.7}`),
	}, {
		name: "handling a list response",
		CEL: &triggersv1.CELInterceptor{
			Overlays: []triggersv1.CELOverlay{
				{Key: "event", Expression: "body.event.map(s, s['testing'])"},
			},
		},
		body: json.RawMessage(`{"event":[{"testing":"value"},{"testing":"another"}]}`),
		wantExtensions: map[string]interface{}{
			"event": []interface{}{"value", "another"},
		},
	}, {
		name: "return different types of expression",
		CEL: &triggersv1.CELInterceptor{
			Overlays: []triggersv1.CELOverlay{
				{Key: "one", Expression: "'yo'"},
				{Key: "two", Expression: "false ? true : false"},
				{Key: "three", Expression: "body.test"},
			},
		},
		body: json.RawMessage(`{"value":"test","test":{"other":"thing"}}`),
		wantExtensions: map[string]interface{}{
			"one": "yo",
			"two": false,
			"three": map[string]interface{}{
				"other": "thing",
			},
		},
	}, {
		name: "demonstrate defaulting logic within cel interceptor",
		CEL: &triggersv1.CELInterceptor{
			Overlays: []triggersv1.CELOverlay{
				{Key: "one", Expression: "has(body.value) ? body.value : 'default'"},
				{Key: "two", Expression: "has(body.test.second) ? body.test.second : 'default'"},
				{Key: "three", Expression: "has(body.test.third) && has(body.test.third.thing) ? body.value.third.thing : 'default'"},
			},
		},
		body: json.RawMessage(`{"value":"test","test":{"other":"thing"}}`),
		wantExtensions: map[string]interface{}{
			"one":   "test",
			"two":   "default",
			"three": "default",
		},
	}, {
		name: "filters and overlays can access passed in extensions",
		CEL: &triggersv1.CELInterceptor{
			Filter: `extensions.foo == "bar"`,
			Overlays: []triggersv1.CELOverlay{
				{Key: "one", Expression: "extensions.foo"},
			},
		},
		extensions: map[string]interface{}{
			"foo": "bar",
		},
		wantExtensions: map[string]interface{}{
			"one": "bar",
		},
	}, {
		name: "decode with cel extension to a field",
		CEL: &triggersv1.CELInterceptor{
			Overlays: []triggersv1.CELOverlay{
				{Key: "value", Expression: "base64.decode(body.b64value) == b'hello'"},
				{Key: "compare_string", Expression: "base64.decode(body.b64value) == bytes('hello')"},
				{Key: "decoded", Expression: "base64.decode(body.b64value)"},
				{Key: "decoded_string", Expression: "string(base64.decode(body.b64value))"},
			},
		},
		body: json.RawMessage(`{"b64value":"aGVsbG8=","test":"hello"}`),
		wantExtensions: map[string]interface{}{
			"value":          true,
			"compare_string": true,
			"decoded":        "aGVsbG8=",
			"decoded_string": "hello"},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(rt *testing.T) {
			ctx, _ := test.SetupFakeContext(t)
			var clientset *fake.Clientset
			ctx, clientset = fakekubeclient.With(ctx, makeSecret())
			w := &Interceptor{
				SecretGetter: interceptors.DefaultSecretGetter(clientset.CoreV1()),
			}
			res := w.Process(ctx, &triggersv1.InterceptorRequest{
				Body: string(tt.body),
				Header: http.Header{
					"Content-Type":   []string{"application/json"},
					"X-Test":         []string{"test-value"},
					"X-Secret-Token": []string{"secrettoken"},
				},
				Extensions: tt.extensions,
				InterceptorParams: map[string]interface{}{
					"filter":   tt.CEL.Filter,
					"overlays": tt.CEL.Overlays,
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: fmt.Sprintf("namespaces/%s/triggers/example-trigger", testNS),
				},
			})
			if !res.Continue {
				rt.Fatalf("cel.Process() unexpectedly returned continue: false. Response is: %v", res.Status.Err())
			}
			if tt.wantExtensions != nil {
				got := res.Extensions
				if diff := cmp.Diff(tt.wantExtensions, got); diff != "" {
					rt.Fatalf("cel.Process() did return correct extensions (-wantMsg+got): %v", diff)
				}
			}
		})
	}
}

func TestInterceptor_Process_Error(t *testing.T) {
	tests := []struct {
		name     string
		CEL      *triggersv1.CELInterceptor
		body     []byte
		wantCode codes.Code
		wantMsg  string
	}{{
		name: "simple body check with non-matching body",
		CEL: &triggersv1.CELInterceptor{
			Filter: "body.value == 'test'",
		},
		body:     []byte(`{"value":"testing"}`),
		wantCode: codes.FailedPrecondition,
		wantMsg:  "expression body.value == 'test' did not return true",
	}, {
		name: "simple header check with non matching header",
		CEL: &triggersv1.CELInterceptor{
			Filter: "header['X-Test'][0] == 'unknown'",
		},
		body:     []byte(`{}`),
		wantCode: codes.FailedPrecondition,
		wantMsg:  "expression header.*'unknown' did not return true",
	}, {
		name: "overloaded header check with case insensitive failed match",
		CEL: &triggersv1.CELInterceptor{
			Filter: "header.match('x-test', 'no-match')",
		},
		body:     []byte(`{}`),
		wantCode: codes.FailedPrecondition,
		wantMsg:  "expression header.match\\('x-test', 'no-match'\\) did not return true",
	}, {
		name: "unable to parse the expression",
		CEL: &triggersv1.CELInterceptor{
			Filter: "header['X-Test",
		},
		body:     []byte(`{"value":"test"}`),
		wantCode: codes.InvalidArgument,
		wantMsg:  "Syntax error: token recognition error at: ''X-Test'",
	}, {
		name: "unable to parse the JSON body",
		CEL: &triggersv1.CELInterceptor{
			Filter: "body.value == 'test'",
		},
		body:     []byte(`{]`),
		wantCode: codes.InvalidArgument,
		wantMsg:  "invalid character ']' looking for beginning of object key string",
	}, {
		name: "bad overlay",
		CEL: &triggersv1.CELInterceptor{
			Filter: "body.value == 'test'",
			Overlays: []triggersv1.CELOverlay{
				{Key: "new", Expression: "test.value"},
			},
		},
		body:     []byte(`{"value":"test"}`),
		wantCode: codes.InvalidArgument,
		wantMsg:  `expression "test.value" check failed: ERROR:.*undeclared reference to 'test'`,
	},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Interceptor{}
			res := w.Process(context.Background(), &triggersv1.InterceptorRequest{
				Body: string(tt.body),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"X-Test":       []string{"test-value"},
				},
				Extensions: nil,
				InterceptorParams: map[string]interface{}{
					"filter":   tt.CEL.Filter,
					"overlays": tt.CEL.Overlays,
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			})
			if res.Continue {
				t.Fatalf("cel.Process() uexpectedly returned continue: true. Response: %+v", res)
			}
			if tt.wantCode != res.Status.Code {
				t.Errorf("cel.Process() unexpected status.Code. wanted: %v, got: %v. Status is: %+v", tt.wantCode, res.Status.Code, res.Status.Err())
			}
			if !checkMessageContains(t, tt.wantMsg, res.Status.Message) {
				t.Fatalf("cel.Process() got %+v, wanted status.message to contain %s", res.Status.Err(), tt.wantMsg)

			}
		})
	}
}

func TestInterceptor_Process_InvalidParams(t *testing.T) {
	w := &Interceptor{}
	res := w.Process(context.Background(), &triggersv1.InterceptorRequest{
		Body:   `{}`,
		Header: http.Header{},
		InterceptorParams: map[string]interface{}{
			"filter": func() {}, // Should fail JSON unmarshal
		},
		Context: &triggersv1.TriggerContext{
			EventURL:  "https://testing.example.com",
			TriggerID: "namespaces/default/triggers/example-trigger",
		},
	})

	if res.Continue {
		t.Fatalf("cel.Process() uexpectedly returned continue: true. Response: %+v", res)
	}
	if codes.InvalidArgument != res.Status.Code {
		t.Errorf("cel.Process() unexpected status.Code. wanted: %v, got: %v. Status is: %+v", codes.InvalidArgument, res.Status.Code, res.Status.Err())
	}
	wantErrMsg := "failed to marshal json"
	if !checkMessageContains(t, wantErrMsg, res.Status.Message) {
		t.Fatalf("cel.Process() got %+v, wanted status.message to contain %s", res.Status.Err(), wantErrMsg)

	}
}

func TestExpressionEvaluation(t *testing.T) {
	reg, err := types.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	testSHA := "ec26c3e57ca3a959ca5aad62de7213c562f8c821"
	testRef := "refs/heads/master"
	jsonMap := map[string]interface{}{
		"value": "testing",
		"sha":   testSHA,
		"ref":   testRef,
		"pull_request": map[string]interface{}{
			"commits": 2,
		},
		"upperMsg":  "THIS IS LOWER CASE",
		"b64value":  "ZXhhbXBsZQ==",
		"json_body": `{"testing": "value", "number": 2}`,
		"yaml_body": "key1: value1\nkey2: value2\nkey3: value3\n",
		"testURL":   "https://user:password@site.example.com/path/to?query=search#first",
		"multiURL":  "https://user:password@site.example.com/path/to?query=search&query=results",
		"jsonObject": map[string]interface{}{
			"string":  "value",
			"integer": 2,
		},
		"jsonArray": []string{
			"one", "two",
		},
		"jsonObjects": []map[string]interface{}{
			{
				"testing1": map[string]interface{}{
					"testing": []string{"test1", "test2"},
				},
			},
		},
	}

	refParts := strings.Split(testRef, "/")
	header := http.Header{}
	header.Add("X-Test-Header", "value")
	req := httptest.NewRequest(http.MethodPost, "https://example.com/testing?param=value", nil)
	evalEnv := map[string]interface{}{"body": jsonMap, "header": header, "requestURL": req.URL.String()}
	tests := []struct {
		name   string
		expr   string
		secret *corev1.Secret
		want   ref.Val
	}{
		{
			name: "simple body value",
			expr: "body.value",
			want: types.String("testing"),
		},
		{
			name: "boolean body value",
			expr: "body.value == 'testing'",
			want: types.True,
		},
		{
			name: "truncate a long string",
			expr: "body.sha.truncate(7)",
			want: types.String("ec26c3e"),
		},
		{
			name: "truncate a string to its own length",
			expr: "body.value.truncate(7)",
			want: types.String("testing"),
		},
		{
			name: "truncate a string to fewer characters than it has",
			expr: "body.sha.truncate(45)",
			want: types.String(testSHA),
		},
		{
			name: "split a string on a character",
			expr: "body.ref.split('/')",
			want: reg.NativeToValue(refParts),
		},
		{
			name: "extract a branch from a non refs string",
			expr: "body.value.split('/')",
			want: reg.NativeToValue([]string{"testing"}),
		},
		{
			name: "combine split and truncate",
			expr: "body.value.split('/')[0].truncate(2)",
			want: types.String("te"),
		},
		{
			name: "exact header lookup",
			expr: "header.canonical('X-Test-Header')",
			want: types.String("value"),
		},
		{
			name: "canonical header lookup",
			expr: "header.canonical('x-test-header')",
			want: types.String("value"),
		},
		{
			name: "increment an integer",
			expr: "body.pull_request.commits + 1",
			want: types.Int(3),
		},
		{
			name:   "compare string against secret",
			expr:   "'secrettoken'.compareSecret('token', 'test-secret', 'testing-ns') ",
			want:   types.True,
			secret: makeSecret(),
		},
		{
			name:   "compare string against secret with no match",
			expr:   "'nomatch'.compareSecret('token', 'test-secret', 'testing-ns') ",
			want:   types.False,
			secret: makeSecret(),
		},
		{
			name:   "compare string against secret in the default namespace",
			expr:   "'secrettoken'.compareSecret('token', 'test-secret') ",
			want:   types.True,
			secret: makeSecret(),
		},
		{
			name: "parse JSON body in a string",
			expr: "body.json_body.parseJSON().testing == 'value'",
			want: types.True,
		},
		{
			name: "compare a JSON number to an integer variable",
			expr: "body.json_body.parseJSON().number == body.jsonObject.integer",
			want: types.True,
		},
		{
			name: "compare a JSON number to an int literal",
			expr: "body.json_body.parseJSON().number > 1",
			want: types.True,
		},
		{
			name: "compare a JSON number to a uint literal",
			expr: "body.json_body.parseJSON().number < 3u",
			want: types.True,
		},
		{
			name: "compare a JSON number to a double literal",
			expr: "body.json_body.parseJSON().number == 2.0",
			want: types.True,
		},
		{
			name: "compare a JSON field to null",
			expr: "body.json_body.parseJSON().number == null",
			want: types.False,
		},
		{
			name: "parse YAML body in a string",
			expr: "body.yaml_body.parseYAML().key1 == 'value1'",
			want: types.True,
		},
		{
			name: "parse URL",
			expr: "body.testURL.parseURL().path == '/path/to'",
			want: types.True,
		},
		{
			name: "parse URL and extract single string",
			expr: "body.testURL.parseURL().query['query'] == 'search'",
			want: types.True,
		},
		{
			name: "parse URL and extract multiple strings",
			expr: "body.multiURL.parseURL().queryStrings['query']",
			want: reg.NativeToValue([]string{"search", "results"}),
		},
		{
			name: "parse request url",
			expr: "requestURL.parseURL().path",
			want: types.String("/testing"),
		},
		{
			name: "lower casing a string",
			expr: "body.upperMsg.lowerAscii()",
			want: types.String("this is lower case"),
		},
		{
			name: "marshal JSON object to string",
			expr: "body.jsonObject.marshalJSON()",
			want: types.String(`{"integer":2,"string":"value"}`),
		},
		{
			name: "marshal JSON array to string",
			expr: "body.jsonArray.marshalJSON()",
			want: types.String(`["one","two"]`),
		},
		{
			name: "marshal JSON objects to string",
			expr: "body.jsonObjects.marshalJSON()",
			want: types.String(`[{"testing1":{"testing":["test1","test2"]}}]`),
		},
		{
			name: "extension base64 decoding",
			expr: "base64.decode(body.b64value)",
			want: types.Bytes("example"),
		},
		{
			name: "extension base64 encoding",
			expr: "base64.encode(b'example')",
			want: types.String("ZXhhbXBsZQ=="),
		},
		{
			name: "extension string join",
			expr: "body.jsonArray.join(', ')",
			want: types.String("one, two"),
		},
		{
			name: "extension string join",
			expr: "body.jsonArray.join(', ')",
			want: types.String("one, two"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(rt *testing.T) {
			ctx, _ := test.SetupFakeContext(rt)
			clientset := fakekubeclient.Get(ctx)
			if tt.secret != nil {
				_, clientset = fakekubeclient.With(ctx, tt.secret)
			}
			env, err := makeCelEnv(context.Background(), testNS, interceptors.DefaultSecretGetter(clientset.CoreV1()))
			if err != nil {
				t.Fatal(err)
			}
			got, err := evaluate(tt.expr, env, evalEnv)
			if err != nil {
				rt.Errorf("evaluate() got an error %s", err)
				return
			}
			_, ok := got.(*types.Err)
			if ok {
				rt.Errorf("error evaluating expression: %s", got)
				return
			}
			v, ok := got.Equal(tt.want).(types.Bool)
			if !ok {
				rt.Errorf("failed to compare got %v, want %v", got, tt.want)
			}
			if ok && v != types.True {
				rt.Errorf("evaluate() = %s, wantMsg %s", got, tt.want)
			}
		})
	}
}

func TestExpressionEvaluation_Error(t *testing.T) {
	testSHA := "ec26c3e57ca3a959ca5aad62de7213c562f8c821"
	jsonMap := map[string]interface{}{
		"value":        "testing",
		"sha":          testSHA,
		"valid_yaml":   "key1: value1\nkey2: value2\n",
		"invalid_yaml": "key1: value1key2: value2\n",
		"pull_request": map[string]interface{}{
			"commits": []string{},
		},
	}
	header := http.Header{}
	evalEnv := map[string]interface{}{"body": jsonMap, "header": header}
	tests := []struct {
		name     string
		expr     string
		secretNS string
		want     string
	}{
		{
			name: "unknown value",
			expr: "body.val",
			want: `expression "body.val" failed to evaluate: no such key: val`,
		},
		{
			name: "invalid syntax",
			expr: "body.value = 'testing'",
			want: `failed to parse expression "body.value = 'testing'"`,
		},
		{
			name: "unknown function",
			expr: "trunca(body.sha, 7)",
			want: "undeclared reference to 'trunca'",
		},
		{
			name: "invalid function overloading with match",
			expr: "body.match('testing', 'test')",
			want: "failed to convert to http.Header",
		},
		{
			name: "invalid function overloading with canonical",
			expr: "body.canonical('testing')",
			want: "failed to convert to http.Header",
		},
		{
			name: "missing secret",
			expr: "'testing'.compareSecret('testing', 'testSecret', 'mytoken')",
			want: "failed to find secret.*testing.*",
		},
		{
			name: "invalid parseJSON body",
			expr: "body.value.parseJSON().test == 'test'",
			want: "invalid character 'e' in literal",
		},
		{
			name: "parseJSON decoding non-string",
			expr: "body.pull_request.parseJSON().test == 'test'",
			want: "no such overload: parseJSON(map)",
		},
		{
			name: "parseYAML decoding non-string",
			expr: "body.pull_request.parseYAML().key1 == 'value1'",
			want: "no such overload: parseYAML(map)",
		},
		{
			name: "unknown key",
			expr: "body.valid_yaml.parseYAML().key3 == 'value3'",
			want: "no such key: key3",
		},
		{
			name: "invalid YAML body",
			expr: "body.invalid_yaml.parseYAML().key1 == 'value1'",
			want: "failed to decode 'key1: value1key2: value2\n' in parseYAML:",
		},
		{
			name: "marshalJSON marshalling string",
			expr: "body.value.marshalJSON()",
			want: "no such overload: marshalJSON(string)",
		},
		{
			name: "has function missing nested key",
			expr: "has(body.pull_request.repository.owner) ? body.pull_request.repository.owner : 'me'",
			want: `failed to evaluate: no such key: repository`,
		},
		{
			name: "truncate json",
			expr: "body.pull_request.truncate(7)",
			want: "no such overload: truncate(map, int)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(rt *testing.T) {
			ctx, _ := test.SetupFakeContext(t)
			ns := testNS
			clientset := fakekubeclient.Get(ctx)
			if tt.secretNS != "" {
				_, clientset = fakekubeclient.With(ctx, makeSecret())
				ns = tt.secretNS
			}
			env, err := makeCelEnv(context.Background(), ns, interceptors.DefaultSecretGetter(clientset.CoreV1()))
			if err != nil {
				t.Fatal(err)
			}
			_, err = evaluate(tt.expr, env, evalEnv)
			if err == nil {
				t.Fatal("evaluate() expected err but got nil.")
			}

			if !checkMessageContains(t, tt.want, err.Error()) && !strings.Contains(err.Error(), tt.want) {
				rt.Errorf("evaluate() got %s, wanted %s", err, tt.want)
			}
		})
	}
}

func TestURLToMap(t *testing.T) {
	u, err := url.Parse("https://user:testing@example.com/search?q=dotnet#first")
	if err != nil {
		t.Fatal(err)
	}
	m := urlToMap(u)
	want := map[string]interface{}{
		"scheme": "https",
		"auth": map[string]string{
			"username": "user",
			"password": "testing",
		},
		"host":         "example.com",
		"path":         "/search",
		"rawQuery":     "q=dotnet",
		"fragment":     "first",
		"query":        map[string]string{"q": "dotnet"},
		"queryStrings": url.Values{"q": {"dotnet"}},
	}

	if diff := cmp.Diff(want, m); diff != "" {
		t.Fatalf("urlToMap failed:\n%s", diff)
	}
}

func TestMakeEvalContextWithError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	payload := []byte(`{"tes`)

	_, err := makeEvalContext(payload, req.Header, req.URL.String(), map[string]interface{}{})

	if err == nil {
		t.Fatalf("makeEvalContext(). expected err was nil")
	}

	if !checkMessageContains(t, "failed to parse the body as JSON: unexpected end of JSON input", err.Error()) {
		t.Fatalf("failed to match the error: %s", err)
	}
}

func checkMessageContains(t *testing.T, x, y string) bool {
	t.Helper()
	match, err := regexp.MatchString(x, y)
	if err != nil {
		t.Fatal(err)
	}
	return match
}

func makeSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS,
			Name:      "test-secret",
		},
		Data: map[string][]byte{
			"token": []byte("secrettoken"),
		},
	}
}
