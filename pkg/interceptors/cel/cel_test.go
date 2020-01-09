package cel

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"
	"testing"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/tektoncd/pipeline/pkg/logging"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

func TestInterceptor_ExecuteTrigger(t *testing.T) {
	tests := []struct {
		name    string
		CEL     *triggersv1.CELInterceptor
		payload io.ReadCloser
		want    []byte
	}{
		{
			name: "simple body check with matching body",
			CEL: &triggersv1.CELInterceptor{
				Filter: "body.value == 'testing'",
			},
			payload: ioutil.NopCloser(bytes.NewBufferString(`{"value":"testing"}`)),
			want:    []byte(`{"value":"testing"}`),
		},
		{
			name: "simple header check with matching header",
			CEL: &triggersv1.CELInterceptor{
				Filter: "header['X-Test'][0] == 'test-value'",
			},
			payload: ioutil.NopCloser(bytes.NewBufferString(`{}`)),
			want:    []byte(`{}`),
		},
		{
			name: "overloaded header check with case insensitive matching",
			CEL: &triggersv1.CELInterceptor{
				Filter: "header.match('x-test', 'test-value')",
			},
			payload: ioutil.NopCloser(bytes.NewBufferString(`{}`)),
			want:    []byte(`{}`),
		},
		{
			name: "body and header check",
			CEL: &triggersv1.CELInterceptor{
				Filter: "header.match('x-test', 'test-value') && body.value == 'test'",
			},
			payload: ioutil.NopCloser(bytes.NewBufferString(`{"value":"test"}`)),
			want:    []byte(`{"value":"test"}`),
		},
		{
			name: "single update record",
			CEL: &triggersv1.CELInterceptor{
				Filter: "body.value == 'test'",
				Overlays: []triggersv1.CELOverlay{
					{Key: "new", Expression: "body.value"},
				},
			},
			payload: ioutil.NopCloser(bytes.NewBufferString(`{"value":"test"}`)),
			want:    []byte(`{"new":"test","value":"test"}`),
		},
		{
			name: "multiple update records",
			CEL: &triggersv1.CELInterceptor{
				Filter: "body.value == 'test'",
				Overlays: []triggersv1.CELOverlay{
					{Key: "test.one", Expression: "body.value"},
					{Key: "test.two", Expression: "body.value"},
				},
			},
			payload: ioutil.NopCloser(bytes.NewBufferString(`{"value":"test"}`)),
			want:    []byte(`{"test":{"two":"test","one":"test"},"value":"test"}`),
		},
		{
			name:    "nil body does not panic",
			CEL:     &triggersv1.CELInterceptor{Filter: "header.match('x-test', 'test-value')"},
			payload: nil,
			want:    []byte(`{}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := logging.NewLogger("", "")
			w := &Interceptor{
				CEL:    tt.CEL,
				Logger: logger,
			}
			request := &http.Request{
				Body: tt.payload,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"X-Test":       []string{"test-value"},
				},
			}
			fmt.Printf("executing test %s\n", tt.name)
			resp, err := w.ExecuteTrigger(request)
			if err != nil {
				t.Errorf("Interceptor.ExecuteTrigger() error = %v", err)
			}
			got, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("error reading response body: %v", err)
			}
			defer resp.Body.Close()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Interceptor.ExecuteTrigger() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestInterceptor_ExecuteTrigger_Errors(t *testing.T) {
	tests := []struct {
		name    string
		CEL     *triggersv1.CELInterceptor
		payload []byte
		want    string
	}{
		{
			name: "simple body check with non-matching body",
			CEL: &triggersv1.CELInterceptor{
				Filter: "body.value == 'test'",
			},
			payload: []byte(`{"value":"testing"}`),
			want:    "expression body.value == 'test' did not return true",
		},
		{
			name: "simple header check with non matching header",
			CEL: &triggersv1.CELInterceptor{
				Filter: "header['X-Test'][0] == 'unknown'",
			},
			payload: []byte(`{}`),
			want:    "expression header.*'unknown' did not return true",
		},
		{
			name: "overloaded header check with case insensitive failed match",
			CEL: &triggersv1.CELInterceptor{
				Filter: "header.match('x-test', 'no-match')",
			},
			payload: []byte(`{}`),
			want:    "expression header.match\\('x-test', 'no-match'\\) did not return true",
		},
		{
			name: "unable to parse the expression",
			CEL: &triggersv1.CELInterceptor{
				Filter: "header['X-Test",
			},
			payload: []byte(`{"value":"test"}`),
			want:    "Syntax error: token recognition error at: ''X-Test'",
		},
		{
			name: "unable to parse the JSON body",
			CEL: &triggersv1.CELInterceptor{
				Filter: "body.value == 'test'",
			},
			payload: []byte(`{]`),
			want:    "invalid character ']' looking for beginning of object key string",
		},
		{
			name: "bad update record",
			CEL: &triggersv1.CELInterceptor{
				Filter: "body.value == 'test'",
				Overlays: []triggersv1.CELOverlay{
					{Key: "new", Expression: "test.value"},
				},
			},
			payload: []byte(`{"value":"test"}`),
			want:    "failed to evaluate overlay expression 'body.value == 'test''",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := logging.NewLogger("", "")
			w := &Interceptor{
				CEL:    tt.CEL,
				Logger: logger,
			}
			request := &http.Request{
				Body: ioutil.NopCloser(bytes.NewReader(tt.payload)),
				GetBody: func() (io.ReadCloser, error) {
					return ioutil.NopCloser(bytes.NewReader(tt.payload)), nil
				},
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"X-Test":       []string{"test-value"},
				},
			}
			_, err := w.ExecuteTrigger(request)
			if !matchError(t, tt.want, err) {
				t.Errorf("evaluate() got %s, wanted %s", err, tt.want)
				return
			}
		})
	}
}

func TestExpressionEvaluation(t *testing.T) {
	testSHA := "ec26c3e57ca3a959ca5aad62de7213c562f8c821"
	jsonMap := map[string]interface{}{
		"value": "testing",
		"sha":   testSHA,
	}
	header := http.Header{}
	evalEnv := map[string]interface{}{"body": jsonMap, "header": header}
	env, err := makeCelEnv()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		expr string
		want ref.Val
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
			expr: "truncate(body.sha, 7)",
			want: types.String("ec26c3e"),
		},
		{
			name: "truncate a string to fewer characters than it has",
			expr: "truncate(body.sha, 45)",
			want: types.String(testSHA),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluate(tt.expr, env, evalEnv)
			if err != nil {
				t.Errorf("evaluate() got an error %s", err)
				return
			}
			_, ok := got.(*types.Err)
			if ok {
				t.Errorf("error evaluating expression: %s", got)
				return
			}

			if !got.Equal(tt.want).(types.Bool) {
				t.Errorf("evaluate() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestExpressionEvaluation_Error(t *testing.T) {
	testSHA := "ec26c3e57ca3a959ca5aad62de7213c562f8c821"
	jsonMap := map[string]interface{}{
		"value": "testing",
		"sha":   testSHA,
	}
	header := http.Header{}
	evalEnv := map[string]interface{}{"body": jsonMap, "headers": header}
	env, err := makeCelEnv()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		expr string
		want string
	}{
		{
			name: "unknown value",
			expr: "body.val",
			want: "no such key: val",
		},
		{
			name: "invalid syntax",
			expr: "body.value = 'testing'",
			want: "Syntax error: token recognition error",
		},
		{
			name: "unknown function",
			expr: "trunca(body.sha, 7)",
			want: "undeclared reference to 'trunca'",
		},
		{
			name: "invalid function overloading",
			expr: "body.match('testing', 'test')",
			want: "failed to convert to http.Header",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := evaluate(tt.expr, env, evalEnv)
			if !matchError(t, tt.want, err) {
				t.Errorf("evaluate() got %s, wanted %s", err, tt.want)
				return
			}
		})
	}
}

func matchError(t *testing.T, s string, e error) bool {
	t.Helper()
	match, err := regexp.MatchString(s, e.Error())
	if err != nil {
		t.Fatal(err)
	}
	return match
}
