package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/google/go-cmp/cmp"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"go.uber.org/zap/zaptest"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestServer_ServeHTTP(t *testing.T) {
	// includes error cases when the error is from interceptor processing.

	testTriggerContext := &v1alpha1.TriggerContext{
		EventURL:  "http://something",
		EventID:   "abcde",
		TriggerID: "namespaces/default/triggers/test-trigger",
	}
	tests := []struct {
		name string
		path string
		req  *v1alpha1.InterceptorRequest
		want *v1alpha1.InterceptorResponse
	}{{
		name: "valid request that should continue",
		path: "/cel",
		req: &v1alpha1.InterceptorRequest{
			Body: json.RawMessage(`{}`),
			Header: map[string][]string{
				"X-Event-Type": {"push"},
			},
			InterceptorParams: map[string]interface{}{
				"filter": "header.canonical(\"X-Event-Type\") == \"push\"",
			},
			Context: testTriggerContext,
		},
		want: &v1alpha1.InterceptorResponse{
			Continue: true,
		},
	}, {
		name: "valid request that should not continue",
		path: "/cel",
		req: &v1alpha1.InterceptorRequest{
			Body: json.RawMessage(`{}`),
			Header: map[string][]string{
				"X-Event-Type": {"push"},
			},
			InterceptorParams: map[string]interface{}{
				"filter": "header.canonical(\"X-Event-Type\") == \"pull\"",
			},
			Context: testTriggerContext,
		},
		want: &v1alpha1.InterceptorResponse{
			Continue: false,
			Status: v1alpha1.Status{
				Code:    codes.FailedPrecondition,
				Message: `expression header.canonical("X-Event-Type") == "pull" did not return true`,
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			ctx, _ := rtesting.SetupFakeContext(t)
			kubeClient := fakekubeclient.Get(ctx)

			server := NewWithCoreInterceptors(kubeClient, logger.Sugar())
			body, err := json.Marshal(tc.req)
			if err != nil {
				t.Fatalf("Failed to marshal errors ")
			}
			req := httptest.NewRequest("POST", fmt.Sprintf("http://example.com%s", tc.path), bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)
			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("ServeHTTP() expected statusCode 200 but got: %d", resp.StatusCode)
			}
			if resp.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("ServeHTTP() expected Content-Type header to be application/json but got: %s", resp.Header.Get("Content-Type"))
			}

			respBody, _ := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
			got := v1alpha1.InterceptorResponse{}
			if err := json.Unmarshal(respBody, &got); err != nil {
				t.Fatalf("ServeHTTP() failed to unmarshal response into struct: %v", err)
			}
			if diff := cmp.Diff(tc.want, &got); diff != "" {
				t.Fatalf("ServeHTTP() response did not match expected. Diff (-want/+got): %s", diff)
			}
		})
	}

}

// Tests unexpected error cases where interceptor processing does not happen.
func TestServer_ServeHTTP_Error(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		req              []byte
		wantResponseCode int
		wantResponseBody string
	}{{
		name:             "bad path",
		path:             "/invalid",
		req:              json.RawMessage(`{}`),
		wantResponseCode: 400,
		wantResponseBody: "path did not match any interceptors",
	}, {
		name:             "invalid body",
		path:             "/cel",
		req:              json.RawMessage(`{}`),
		wantResponseCode: 400,
		wantResponseBody: "failed to parse body as InterceptorRequest",
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			ctx, _ := rtesting.SetupFakeContext(t)
			kubeClient := fakekubeclient.Get(ctx)

			server := NewWithCoreInterceptors(kubeClient, logger.Sugar())
			body, err := json.Marshal(tc.req)
			if err != nil {
				t.Fatalf("Failed to marshal errors ")
			}
			req := httptest.NewRequest("POST", fmt.Sprintf("http://example.com%s", tc.path), bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)
			resp := w.Result()
			if resp.StatusCode != tc.wantResponseCode {
				t.Fatalf("ServeHTTP() expected statusCode %d but got: %d", tc.wantResponseCode, resp.StatusCode)
			}

			respBody, _ := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
			if !strings.Contains(string(respBody), tc.wantResponseBody) {
				t.Fatalf("ServeHTTP() expected response to contain : %s \n but got %s: ", tc.wantResponseBody, string(respBody))
			}
		})
	}
}
