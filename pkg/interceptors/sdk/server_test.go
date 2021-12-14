package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/tektoncd/triggers/pkg/interceptors/cel"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"google.golang.org/grpc/codes"
	"k8s.io/client-go/kubernetes"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc/codes"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/test"
)

func TestServer_ServeHTTP(t *testing.T) {
	// includes error cases when the error is from interceptor processing.

	testTriggerContext := &v1beta1.TriggerContext{
		EventURL:  "http://something",
		EventID:   "abcde",
		TriggerID: "namespaces/default/triggers/test-trigger",
	}
	tests := []struct {
		name string
		path string
		req  *v1beta1.InterceptorRequest
		want *v1beta1.InterceptorResponse
	}{{
		name: "valid request that should continue",
		path: "/cel",
		req: &v1beta1.InterceptorRequest{
			Body: `{}`,
			Header: map[string][]string{
				"X-Event-Type": {"push"},
			},
			InterceptorParams: map[string]interface{}{
				"filter": "header.canonical(\"X-Event-Type\") == \"push\"",
			},
			Context: testTriggerContext,
		},
		want: &v1beta1.InterceptorResponse{
			Continue: true,
		},
	}, {
		name: "valid request that should not continue",
		path: "/cel",
		req: &v1beta1.InterceptorRequest{
			Body: `{}`,
			Header: map[string][]string{
				"X-Event-Type": {"push"},
			},
			InterceptorParams: map[string]interface{}{
				"filter": "header.canonical(\"X-Event-Type\") == \"pull\"",
			},
			Context: testTriggerContext,
		},
		want: &v1beta1.InterceptorResponse{
			Continue: false,
			Status: v1beta1.Status{
				Code:    codes.FailedPrecondition,
				Message: `expression header.canonical("X-Event-Type") == "pull" did not return true`,
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := test.SetupFakeContext(t)

			//secretLister := fakeSecretInformer.Get(ctx).Lister()

			server, err := NewWithInterceptors(ctx, map[string]InterceptorFunc{
				"cel": cel.NewInterceptor,
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
			got := v1beta1.InterceptorResponse{}
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
			ctx, _ := test.SetupFakeContext(t)

			//secretLister := fakeSecretInformer.Get(ctx).Lister()

			server, err := NewWithInterceptors(ctx, map[string]InterceptorFunc {
				"cel": cel.NewInterceptor,
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

//type fakeInterceptor struct{}
//
//func (i fakeInterceptor) Process(ctx context.Context, r *v1beta1.InterceptorRequest) *v1beta1.InterceptorResponse {
//	return nil
//}
//
//// TODO: Probably an easier way to do this
//func newFake(ctx context.Context) v1beta1.InterceptorInterface {
//	return fakeInterceptor{}
//}

//func TestServer_RegisterInterceptor(t *testing.T) {
//	s, err := NewWithInterceptors(context.Background(), map[string]InterceptorFunc{
//		"first": newFake,
//	})
//	want := map[string]v1beta1.InterceptorInterface{
//		"first": fakeInterceptor{},
//	}
//	if diff := cmp.Diff(want, s.interceptors); diff != "" {
//		t.Errorf("RegisterInterceptor first (-want/+got): %s", diff)
//	}
//
//	s.RegisterInterceptor("second", fakeInterceptor{})
//	want["second"] = fakeInterceptor{}
//	if diff := cmp.Diff(want, s.interceptors); diff != "" {
//		t.Errorf("RegisterInterceptor second (-want/+got): %s", diff)
//	}
//}
