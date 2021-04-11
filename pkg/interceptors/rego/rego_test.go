package rego

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"google.golang.org/grpc/codes"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	rtesting "knative.dev/pkg/reconciler/testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap/zaptest"
)

var defaultHTTPHeader = http.Header{
	"Content-Type":   []string{"application/json"},
	"X-Test":         []string{"Test-Value"},
	"X-Secret-Token": []string{"secrettoken"},
}
var emptyBody = string(`{}`)
var simpleBody = string(json.RawMessage(`{"value":"testing"}`))

func TestInterceptor_Process(t *testing.T) {
	tests := []struct {
		name               string
		interceptorRequest *triggersv1.InterceptorRequest
		wantExtensions     map[string]interface{}
	}{{
		name: "simple body check with matching body",
		interceptorRequest: &triggersv1.InterceptorRequest{
			InterceptorParams: map[string]interface{}{
				"query": `filter {
					input.body.value = "testing"
				}`,
			},
			Body: simpleBody,
		},
	}, {
		name: "simple header check with case-insensitive matching",
		interceptorRequest: &triggersv1.InterceptorRequest{
			InterceptorParams: map[string]interface{}{
				"query": `filter {
					lower(input.header[q][k]) = "test-value"
					lower(q) = "x-test"
				}`,
			},
			Header: defaultHTTPHeader,
			Body:   emptyBody,
		},
	}, {
		name: "body and header check",
		interceptorRequest: &triggersv1.InterceptorRequest{
			InterceptorParams: map[string]interface{}{
				"query": `filter {
					input.body.value = "testing"
				}`,
			},
			Header: defaultHTTPHeader,
			Body:   simpleBody,
		},
	}, {
		name: "single overlay",
		interceptorRequest: &triggersv1.InterceptorRequest{
			InterceptorParams: map[string]interface{}{
				"query": `filter {
					input.body.value = "testing"
				}`,
				"overlays": []map[string]interface{}{{
					QueryParam:   "input.body.value = result",
					ExtensionKey: "new",
					Bindings:     []string{"result"},
				}},
			},
			Body: simpleBody,
		},
		wantExtensions: map[string]interface{}{
			"new": map[string]interface{}{
				"result": []interface{}{string("testing")},
			},
		},
	}, {
		name: "single result overlay with empty filter",
		interceptorRequest: &triggersv1.InterceptorRequest{
			InterceptorParams: map[string]interface{}{
				// empty filter query
				"query": `filter {true}`,
				"overlays": []map[string]interface{}{{
					QueryParam:   `a = split(input.body.ref, "/"); result = a[i]; i = 2`,
					ExtensionKey: "new",
					Bindings:     []string{"result"},
					Single:       true,
				}},
			},
			Body: string(json.RawMessage(`{"ref":"refs/head/master","name":"testing"}`)),
		},
		wantExtensions: map[string]interface{}{
			"new": map[string]interface{}{
				"result": "master",
			},
		},
	}, {
		name: "multiple overlays",
		interceptorRequest: &triggersv1.InterceptorRequest{
			InterceptorParams: map[string]interface{}{
				// empty filter query
				"query": `filter {
					input.body.value = "testing"
				}`,
				"overlays": []map[string]interface{}{{
					QueryParam:   `input.body.value = one; input.body.value = two`,
					ExtensionKey: "test",
					Bindings:     []string{"one", "two"},
					Single:       true,
				}},
			},
			Body: simpleBody,
		},
		wantExtensions: map[string]interface{}{
			"test": map[string]interface{}{
				"two": "testing",
				"one": "testing",
			},
		},
	}, {
		name: "rego query can access extensions",
		interceptorRequest: &triggersv1.InterceptorRequest{
			InterceptorParams: map[string]interface{}{
				"query": `filter {
					input.extensions.foo = "bar"
				}`,
				"overlays": []map[string]interface{}{{
					QueryParam:   "input.extensions.foo = result",
					ExtensionKey: "new",
					Bindings:     []string{"result"},
				}},
			},
			Extensions: map[string]interface{}{
				"foo": "bar",
			},
			Body: emptyBody,
		},
		wantExtensions: map[string]interface{}{
			"new": map[string]interface{}{
				"result": []interface{}{string("bar")},
			},
		},
	},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(rt *testing.T) {
			logger := zaptest.NewLogger(t)
			ctx, _ := rtesting.SetupFakeContext(t)
			kubeClient := fakekubeclient.Get(ctx)
			w := &Interceptor{
				KubeClientSet: kubeClient,
				Logger:        logger.Sugar(),
			}
			res := w.Process(ctx, tt.interceptorRequest)
			if !res.Continue {
				rt.Fatalf("rego.Process() unexpectedly returned continue: false. Response is: %v", res.Status.Err())
			}
			if tt.wantExtensions != nil {
				got := res.Extensions
				if diff := cmp.Diff(tt.wantExtensions, got); diff != "" {
					rt.Fatalf("rego.Process() did return correct extensions (-wantMsg+got): %v", diff)
				}
			}
		})
	}
}

func TestInterceptor_Process_Error(t *testing.T) {
	tests := []struct {
		name               string
		interceptorRequest *triggersv1.InterceptorRequest
		wantCode           codes.Code
		wantMsg            string
	}{{
		name: "simple body check with non-matching body",
		interceptorRequest: &triggersv1.InterceptorRequest{
			InterceptorParams: map[string]interface{}{
				"query": `filter {
					input.body.value = "test"
				}`,
			},
			Body: simpleBody,
		},
		wantCode: codes.FailedPrecondition,
		wantMsg:  "unmatched",
	}, {
		name: "simple header check with non matching header",
		interceptorRequest: &triggersv1.InterceptorRequest{
			Header: defaultHTTPHeader,
			Body:   emptyBody,
			InterceptorParams: map[string]interface{}{
				"query": `filter {
					lower(input.header[q][k]) = "unknown"
					lower(q) = "x-test"
				}`,
			},
		},
		wantCode: codes.FailedPrecondition,
		wantMsg:  "unmatched",
	}, {
		name: "invalid rego query",
		interceptorRequest: &triggersv1.InterceptorRequest{
			Header: defaultHTTPHeader,
			Body:   emptyBody,
			InterceptorParams: map[string]interface{}{
				"query": `filter {
					lower(header[q][k]) = "unknown"
					lower(q) = "x-test"
				}`,
			},
		},
		wantCode: codes.Aborted,
		wantMsg:  "header is unsafe",
	}, {
		name: "rego filter requires body to be passed in",
		interceptorRequest: &triggersv1.InterceptorRequest{
			InterceptorParams: map[string]interface{}{
				"query": `filter {true}`,
			},
		},
		wantCode: codes.Aborted,
		wantMsg:  "unable to marshal body to json",
	},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			w := &Interceptor{
				Logger: logger.Sugar(),
			}
			res := w.Process(context.Background(), tt.interceptorRequest)
			if res.Continue {
				t.Fatalf("rego.Process() uexpectedly returned continue: true. Response: %+v", res)
			}
			if tt.wantCode != res.Status.Code {
				t.Errorf("rego.Process() unexpected status.Code. wanted: %v, got: %v. Status is: %+v", tt.wantCode, res.Status.Code, res.Status.Err())
			}
			if !checkMessageContains(t, tt.wantMsg, res.Status.Message) {
				t.Fatalf("rego.Process() got %+v, wanted status.message to contain %s", res.Status.Err(), tt.wantMsg)

			}
		})
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
