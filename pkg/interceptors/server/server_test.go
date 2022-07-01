package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	faketriggersclient "github.com/tektoncd/triggers/pkg/client/injection/client/fake"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/test"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	"knative.dev/pkg/system"
)

const (
	second = 6 * time.Second
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
			Body: `{}`,
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
			Body: `{}`,
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
			ctx, _ := test.SetupFakeContext(t)

			server, err := NewWithCoreInterceptors(interceptors.DefaultSecretGetter(fakekubeclient.Get(ctx).CoreV1()), logger.Sugar())
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
		path:             "/cel",
		req:              json.RawMessage(`{}`),
		wantResponseCode: 400,
		wantResponseBody: "failed to parse body as InterceptorRequest",
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			ctx, _ := test.SetupFakeContext(t)

			server, err := NewWithCoreInterceptors(interceptors.DefaultSecretGetter(fakekubeclient.Get(ctx).CoreV1()), logger.Sugar())
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

type fakeInterceptor struct{}

func (i fakeInterceptor) Process(ctx context.Context, r *v1beta1.InterceptorRequest) *v1beta1.InterceptorResponse {
	return nil
}

func TestServer_RegisterInterceptor(t *testing.T) {
	s := Server{}
	s.RegisterInterceptor("first", fakeInterceptor{})
	want := map[string]v1beta1.InterceptorInterface{
		"first": fakeInterceptor{},
	}
	if diff := cmp.Diff(want, s.interceptors); diff != "" {
		t.Errorf("RegisterInterceptor first (-want/+got): %s", diff)
	}

	s.RegisterInterceptor("second", fakeInterceptor{})
	want["second"] = fakeInterceptor{}
	if diff := cmp.Diff(want, s.interceptors); diff != "" {
		t.Errorf("RegisterInterceptor second (-want/+got): %s", diff)
	}
}

func Test_SecretNotExist(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx, _ := test.SetupFakeContext(t)
	clientSet := fakekubeclient.Get(ctx).CoreV1()
	_, _, err := CreateCerts(ctx, clientSet, time.Now().Add(Decade), logger.Sugar())
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Error(err)
	}
}

func createSecret(t *testing.T, noAfter time.Time) (v1.CoreV1Interface, []byte, []byte, error) {
	if err := os.Setenv(InterceptorTLSSvcKey, "testsvc"); err != nil {
		return nil, []byte{}, []byte{}, err
	}
	if err := os.Setenv(InterceptorTLSSecretKey, "testsecrets"); err != nil {
		return nil, []byte{}, []byte{}, err
	}
	if err := os.Setenv("SYSTEM_NAMESPACE", "testns"); err != nil {
		return nil, []byte{}, []byte{}, err
	}
	namespace := system.Namespace()

	logger := zaptest.NewLogger(t)
	ctx, _ := test.SetupFakeContext(t)
	clientSet := fakekubeclient.Get(ctx).CoreV1()
	localObject := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      os.Getenv(InterceptorTLSSecretKey),
			Namespace: namespace,
		},
	}
	if _, err := clientSet.Secrets(namespace).Create(ctx, localObject, metav1.CreateOptions{}); err != nil {
		t.Error(err)
	}
	sCert, caCert, err := CreateCerts(ctx, clientSet, noAfter, logger.Sugar())
	return clientSet, sCert, caCert, err
}

func Test_CreateSecret(t *testing.T) {
	_, sCert, caCert, err := createSecret(t, time.Now().Add(Decade))
	if err != nil {
		t.Error(err)
	}
	if len(sCert) == 0 && len(caCert) == 0 {
		t.Error("expected serverCert and caCert to be created")
	}
}

func Test_UpdateCRDWithCaCert(t *testing.T) {
	ctx, _ := test.SetupFakeContext(t)
	logger := zaptest.NewLogger(t)
	_, sCert, caCert, err := createSecret(t, time.Now().Add(Decade))
	if err != nil {
		t.Error(err)
	}
	if len(sCert) == 0 && len(caCert) == 0 {
		t.Error("expected serverCert and caCert to be created")
	}
	server, err := NewWithCoreInterceptors(interceptors.DefaultSecretGetter(fakekubeclient.Get(ctx).CoreV1()), logger.Sugar())
	if err != nil {
		t.Fatalf("error initializing core interceptors: %v", err)
	}
	server.RegisterInterceptor("firstci", fakeInterceptor{})
	ci := v1alpha1.ClusterInterceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "firstci",
		},
		Spec: v1alpha1.ClusterInterceptorSpec{
			ClientConfig: v1alpha1.ClientConfig{
				CaBundle: nil,
				URL:      nil,
				Service:  nil,
			},
		},
	}
	var ciList []v1alpha1.ClusterInterceptor
	ciList = append(ciList, ci)
	if _, err := faketriggersclient.Get(ctx).TriggersV1alpha1().ClusterInterceptors().Create(ctx, &ci, metav1.CreateOptions{}); err != nil {
		t.Error(err)
	}

	if err := server.updateCRDWithCaCert(ctx, faketriggersclient.Get(ctx).TriggersV1alpha1(), ciList, caCert); err != nil {
		t.Error(err)
	}

	ciNew, err := faketriggersclient.Get(ctx).TriggersV1alpha1().ClusterInterceptors().Get(ctx, "firstci", metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}
	if string(ciNew.Spec.ClientConfig.CaBundle) == "" {
		t.Error("caBundle should exist after successful clusterinterceptor creation")
	}
}

func Test_CheckCertValidity(t *testing.T) {
	ctx, _ := test.SetupFakeContext(t)
	logger := zaptest.NewLogger(t)
	clientSet, sCert, caCert, err := createSecret(t, time.Now().Add(second))
	if err != nil {
		t.Error(err)
	}
	if len(sCert) == 0 && len(caCert) == 0 {
		t.Error("expected serverCert and caCert to be created")
	}

	tc := faketriggersclient.Get(ctx)

	server, err := NewWithCoreInterceptors(interceptors.DefaultSecretGetter(fakekubeclient.Get(ctx).CoreV1()), logger.Sugar())
	if err != nil {
		t.Fatalf("error initializing core interceptors: %v", err)
	}
	server.RegisterInterceptor("firstciforgoroutine", fakeInterceptor{})

	ci := v1alpha1.ClusterInterceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "firstciforgoroutine",
		},
		Spec: v1alpha1.ClusterInterceptorSpec{
			ClientConfig: v1alpha1.ClientConfig{
				CaBundle: nil,
				URL:      nil,
				Service:  nil,
			},
		},
	}
	var ciList []v1alpha1.ClusterInterceptor
	ciList = append(ciList, ci)
	if _, err := faketriggersclient.Get(ctx).TriggersV1alpha1().ClusterInterceptors().Create(ctx, &ci, metav1.CreateOptions{}); err != nil {
		t.Error(err)
	}

	if err := server.updateCRDWithCaCert(ctx, faketriggersclient.Get(ctx).TriggersV1alpha1(), ciList, caCert); err != nil {
		t.Error(err)
	}

	ciNew, err := faketriggersclient.Get(ctx).TriggersV1alpha1().ClusterInterceptors().Get(ctx, "firstciforgoroutine", metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}

	server.CheckCertValidity(ctx, sCert, caCert, clientSet, logger.Sugar(), tc.TriggersV1alpha1(), 1*time.Second)

	time.Sleep(10 * time.Second)
	ciNew1, err := faketriggersclient.Get(ctx).TriggersV1alpha1().ClusterInterceptors().Get(ctx, "firstciforgoroutine", metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}
	// Making sure that certs expired and and generated one is different than old certs
	if string(ciNew1.Spec.ClientConfig.CaBundle) == "" || string(ciNew1.Spec.ClientConfig.CaBundle) == string(ciNew.Spec.ClientConfig.CaBundle) {
		t.Error("timeout or failed to regenerate certificate after the certificate expire")
	}
}
