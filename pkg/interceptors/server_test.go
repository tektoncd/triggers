package interceptors

import (
	"bytes"
	"context"
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
		path: "/test",
		req: &v1alpha1.InterceptorRequest{
			Body: func() string {
				out, _ := json.Marshal(&v1alpha1.InterceptorResponse{
					Continue: true,
				})
				return string(out)
			}(),
			Header:  map[string][]string{},
			Context: testTriggerContext,
		},
		want: &v1alpha1.InterceptorResponse{
			Continue: true,
		},
	}, {
		name: "valid request that should not continue",
		path: "/test",
		req: &v1alpha1.InterceptorRequest{
			Body: func() string {
				out, _ := json.Marshal(&v1alpha1.InterceptorResponse{
					Continue: false,
					Status: v1alpha1.Status{
						Code:    codes.FailedPrecondition,
						Message: `failed response`,
					},
				})
				return string(out)
			}(),
			Header:            map[string][]string{},
			InterceptorParams: map[string]interface{}{},
			Context:           testTriggerContext,
		},
		want: &v1alpha1.InterceptorResponse{
			Continue: false,
			Status: v1alpha1.Status{
				Code:    codes.FailedPrecondition,
				Message: `failed response`,
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			ctx, _ := rtesting.SetupFakeContext(t)
			kubeClient := fakekubeclient.Get(ctx)

			server, err := NewWithInterceptors(ctx, kubeClient, logger.Sugar(), map[string]v1alpha1.InterceptorInterface{
				"test": &FakeInterceptor{},
			})
			if err != nil {
				t.Fatalf("error initializing core interceptors: %v", err)
			}
			body, err := json.Marshal(tc.req)
			if err != nil {
				t.Fatalf("Failed to marshal errors")
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
		path:             "/test",
		req:              json.RawMessage(`{}`),
		wantResponseCode: 400,
		wantResponseBody: "failed to parse body as InterceptorRequest",
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			ctx, _ := rtesting.SetupFakeContext(t)
			kubeClient := fakekubeclient.Get(ctx)

			server, err := NewWithInterceptors(ctx, kubeClient, logger.Sugar(), map[string]v1alpha1.InterceptorInterface{
				"test": &FakeInterceptor{},
			})
			if err != nil {
				t.Fatalf("error initializing core interceptors: %v", err)
			}
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

type FakeInterceptor struct{}

func (w *FakeInterceptor) Process(ctx context.Context, r *v1alpha1.InterceptorRequest) *v1alpha1.InterceptorResponse {
	response := &v1alpha1.InterceptorResponse{}
	err := json.Unmarshal([]byte(r.Body), response)
	if err != nil {
		return &v1alpha1.InterceptorResponse{
			Continue: false,
			Status: v1alpha1.Status{
				Message: err.Error(),
				Code:    codes.FailedPrecondition,
			},
		}
	}
	return response
}

func (w *FakeInterceptor) InitializeContext(ctx context.Context) {}
