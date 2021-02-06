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

	"go.uber.org/zap/zaptest"

	"google.golang.org/grpc/codes"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	rtesting "knative.dev/pkg/reconciler/testing"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
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
		name: "decodeB64 with parseJSON",
		CEL: &triggersv1.CELInterceptor{
			Overlays: []triggersv1.CELOverlay{
				{Key: "value", Expression: "body.value.decodeb64().parseJSON().test"},
			},
		},
		body: json.RawMessage(`{"value":"eyJ0ZXN0IjoiZGVjb2RlIn0="}`),
		wantExtensions: map[string]interface{}{
			"value": "decode",
		},
	}, {
		name: "decodeB64 to a field",
		CEL: &triggersv1.CELInterceptor{
			Overlays: []triggersv1.CELOverlay{
				{Key: "value", Expression: "body.value.decodeb64()"},
			},
		},
		body:           json.RawMessage(`{"value":"eyJ0ZXN0IjoiZGVjb2RlIn0="}`),
		wantExtensions: map[string]interface{}{"value": "{\"test\":\"decode\"}"},
	}, {
		name: "decode base64 string",
		CEL: &triggersv1.CELInterceptor{
			Overlays: []triggersv1.CELOverlay{
				{Key: "value", Expression: "body.value.decodeb64()"},
			},
		},
		body:           json.RawMessage(`{"value":"dGVzdGluZw=="}`),
		wantExtensions: map[string]interface{}{"value": "testing"},
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
		// TODO: Fix extensions iff key contains ., use sjson to m	erge
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
	},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(rt *testing.T) {
			logger := zaptest.NewLogger(t)
			ctx, _ := rtesting.SetupFakeContext(t)
			kubeClient := fakekubeclient.Get(ctx)
			if _, err := kubeClient.CoreV1().Secrets(testNS).Create(ctx, makeSecret(), metav1.CreateOptions{}); err != nil {
				rt.Error(err)
			}
			w := &Interceptor{
				KubeClientSet: kubeClient,
				Logger:        logger.Sugar(),
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
			logger := zaptest.NewLogger(t)
			w := &Interceptor{
				Logger: logger.Sugar(),
			}
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
	logger := zaptest.NewLogger(t)
	w := &Interceptor{
		Logger: logger.Sugar(),
	}
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
	if codes.InvalidArgument != codes.Code(res.Status.Code) {
		t.Errorf("cel.Process() unexpected status.Code. wanted: %v, got: %v. Status is: %+v", codes.InvalidArgument, res.Status.Code, res.Status.Err())
	}
	wantErrMsg := "failed to marshal json"
	if !checkMessageContains(t, wantErrMsg, res.Status.Message) {
		t.Fatalf("cel.Process() got %+v, wanted status.message to contain %s", res.Status.Err(), wantErrMsg)

	}
}

func TestExpressionEvaluation(t *testing.T) {
	reg := types.NewRegistry()
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
		"json_body": `{"testing": "value"}`,
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
			want: types.Bool(true),
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
			want: types.NewStringList(types.NewRegistry(), refParts),
		},
		{
			name: "extract a branch from a non refs string",
			expr: "body.value.split('/')",
			want: types.NewStringList(types.NewRegistry(), []string{"testing"}),
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
			name: "decode a base64 value",
			expr: "body.b64value.decodeb64()",
			want: types.String("example"),
		},
		{
			name: "increment an integer",
			expr: "body.pull_request.commits + 1",
			want: types.Int(3),
		},
		{
			name:   "compare string against secret",
			expr:   "'secrettoken'.compareSecret('token', 'test-secret', 'testing-ns') ",
			want:   types.Bool(true),
			secret: makeSecret(),
		},
		{
			name:   "compare string against secret with no match",
			expr:   "'nomatch'.compareSecret('token', 'test-secret', 'testing-ns') ",
			want:   types.Bool(false),
			secret: makeSecret(),
		},
		{
			name:   "compare string against secret in the default namespace",
			expr:   "'secrettoken'.compareSecret('token', 'test-secret') ",
			want:   types.Bool(true),
			secret: makeSecret(),
		},
		{
			name: "parse JSON body in a string",
			expr: "body.json_body.parseJSON().testing == 'value'",
			want: types.Bool(true),
		},
		{
			name: "parse YAML body in a string",
			expr: "body.yaml_body.parseYAML().key1 == 'value1'",
			want: types.Bool(true),
		},
		{
			name: "parse URL",
			expr: "body.testURL.parseURL().path == '/path/to'",
			want: types.Bool(true),
		},
		{
			name: "parse URL and extract single string",
			expr: "body.testURL.parseURL().query['query'] == 'search'",
			want: types.Bool(true),
		},
		{
			name: "parse URL and extract multiple strings",
			expr: "body.multiURL.parseURL().queryStrings['query']",
			want: types.NewStringList(reg, []string{"search", "results"}),
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(rt *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(rt)
			kubeClient := fakekubeclient.Get(ctx)
			if tt.secret != nil {
				if _, err := kubeClient.CoreV1().Secrets(tt.secret.ObjectMeta.Namespace).Create(ctx, tt.secret, metav1.CreateOptions{}); err != nil {
					rt.Error(err)
				}
			}
			env, err := makeCelEnv(testNS, kubeClient)
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
			if !got.Equal(tt.want).(types.Bool) {
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
			name: "invalid base64 decoding",
			expr: "\"AA=A\".decodeb64()",
			want: "failed to decode 'AA=A' in decodeB64.*illegal base64 data",
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
			name: "base64 decoding non-string",
			expr: "body.pull_request.decodeb64()",
			want: "unexpected type 'map' passed to decodeB64",
		},
		{
			name: "parseJSON decoding non-string",
			expr: "body.pull_request.parseJSON().test == 'test'",
			want: "unexpected type 'map' passed to parseJSON",
		},
		{
			name: "parseYAML decoding non-string",
			expr: "body.pull_request.parseYAML().key1 == 'value1'",
			want: "unexpected type 'map' passed to parseYAML",
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
			want: "unexpected type 'string' passed to marshalJSON",
		},
		{
			name: "has function missing nested key",
			expr: "has(body.pull_request.repository.owner) ? body.pull_request.repository.owner : 'me'",
			want: `failed to evaluate: no such key: repository`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(rt *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			kubeClient := fakekubeclient.Get(ctx)
			ns := testNS
			if tt.secretNS != "" {
				secret := makeSecret()
				if _, err := kubeClient.CoreV1().Secrets(secret.ObjectMeta.Namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
					rt.Error(err)
				}
				ns = tt.secretNS
			}
			env, err := makeCelEnv(ns, kubeClient)
			if err != nil {
				t.Fatal(err)
			}
			_, err = evaluate(tt.expr, env, evalEnv)
			if err == nil {
				t.Fatal("evaluate() expected err but got nil.")
			}

			if !checkMessageContains(t, tt.want, err.Error()) {
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
