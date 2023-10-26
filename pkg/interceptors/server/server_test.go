package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	fakesecretinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/secret"
	"knative.dev/pkg/system"
	certresources "knative.dev/pkg/webhook/certificates/resources"
)

const (
	second = 6 * time.Second
)

var (
	testsvc     = "testsvc"
	testsecrets = "testsecrets"
	testns      = "testns"
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

			respBody, _ := io.ReadAll(resp.Body)
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

			respBody, _ := io.ReadAll(resp.Body)
			defer resp.Body.Close()
			if !strings.Contains(string(respBody), tc.wantResponseBody) {
				t.Fatalf("ServeHTTP() expected response to contain : %s \n but got %s: ", tc.wantResponseBody, string(respBody))
			}
		})
	}
}

type fakeInterceptor struct{}

// revive:disable:unused-parameter

func (i fakeInterceptor) Process(ctx context.Context, r *v1beta1.InterceptorRequest) *v1beta1.InterceptorResponse {
	return nil
}

// revive:enable:unused-parameter

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
	_, _, err := createCerts(ctx, clientSet, time.Now().Add(Decade), logger.Sugar(), false)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Error(err)
	}
}

func createSecret(t *testing.T, noAfter time.Time, certExpire bool) (v1.CoreV1Interface, []byte, []byte, error) {
	t.Setenv(interceptorTLSSvcKey, testsvc)
	t.Setenv(interceptorTLSSecretKey, testsecrets)
	t.Setenv("SYSTEM_NAMESPACE", testns)
	namespace := system.Namespace()

	logger := zaptest.NewLogger(t)
	ctx, _ := test.SetupFakeContext(t)
	clientSet := fakekubeclient.Get(ctx).CoreV1()
	localObject := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      os.Getenv(interceptorTLSSecretKey),
			Namespace: namespace,
		},
	}
	if _, err := clientSet.Secrets(namespace).Create(ctx, localObject, metav1.CreateOptions{}); err != nil {
		t.Error(err)
	}
	sCert, caCert, err := createCerts(ctx, clientSet, noAfter, logger.Sugar(), certExpire)
	return clientSet, sCert, caCert, err
}

func Test_CreateSecret(t *testing.T) {
	_, sCert, caCert, err := createSecret(t, time.Now().Add(Decade), true)
	if err != nil {
		t.Error(err)
	}
	if len(sCert) == 0 && len(caCert) == 0 {
		t.Error("expected serverCert and caCert to be created")
	}
}

func Test_CheckCertValidity(t *testing.T) {
	ctx, _ := test.SetupFakeContext(t)
	logger := zaptest.NewLogger(t)
	clientSet, sCert, caCert, err := createSecret(t, time.Now().Add(second), false)
	if err != nil {
		t.Error(err)
	}
	if len(sCert) == 0 && len(caCert) == 0 {
		t.Error("expected serverCert and caCert to be created")
	}

	tc := faketriggersclient.Get(ctx)
	server, ci := registerAndGetCI(ctx, t, "firstciforgoroutine", logger)

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

	server.checkCertValidity(ctx, sCert, caCert, clientSet, logger.Sugar(), tc.TriggersV1alpha1(), time.Second)

	time.Sleep(10 * time.Second)
	ciNew1, err := faketriggersclient.Get(ctx).TriggersV1alpha1().ClusterInterceptors().Get(ctx, "firstciforgoroutine", metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}
	// Making sure that certs expired and generated one is different than old certs
	if string(ciNew1.Spec.ClientConfig.CaBundle) == "" || string(ciNew1.Spec.ClientConfig.CaBundle) == string(ciNew.Spec.ClientConfig.CaBundle) {
		t.Error("timeout or failed to regenerate certificate after the certificate expire")
	}
}

func Test_CreateAndValidateCerts(t *testing.T) {
	ctx, _ := test.SetupFakeContext(t)
	logger := zaptest.NewLogger(t)
	clientSet := fakekubeclient.Get(ctx).CoreV1()
	tc := faketriggersclient.Get(ctx)
	t.Setenv(interceptorTLSSecretKey, testsecrets)

	createSecretWithData(ctx, t, clientSet)

	server, ci := registerAndGetCI(ctx, t, "firstci", logger)
	if _, err := faketriggersclient.Get(ctx).TriggersV1alpha1().ClusterInterceptors().Create(ctx, &ci, metav1.CreateOptions{}); err != nil {
		t.Error(err)
	}

	CreateAndValidateCerts(ctx, clientSet, logger.Sugar(), server, tc.TriggersV1alpha1())

	ciNew, err := faketriggersclient.Get(ctx).TriggersV1alpha1().ClusterInterceptors().Get(ctx, "firstci", metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}

	if string(ciNew.Spec.ClientConfig.CaBundle) == "" {
		t.Error("caBundle should exist after successful clusterinterceptor creation")
	}
}

func Test_GetTLSData(t *testing.T) {
	ctx, _ := test.SetupFakeContext(t)
	logger := zaptest.NewLogger(t)
	secretInformer := fakesecretinformer.Get(ctx)
	clientSet := fakekubeclient.Get(ctx).CoreV1()
	serverKey, serverCert, caCert, namespace := getCerts(ctx, t)
	tests := []struct {
		name       string
		secretName string
		secretData map[string][]byte
		want       string
	}{
		{
			name:       "Invalid secret",
			secretName: "test111",
			want:       "secret \"test111\" not found",
		},
		{
			name:       "missing key",
			secretName: testsecrets,
			secretData: map[string][]byte{
				certresources.ServerCert: serverCert,
				certresources.CACert:     caCert,
			},
			want: "server key missing",
		},
		{
			name:       "missing cert",
			secretName: testsecrets,
			secretData: map[string][]byte{
				certresources.ServerKey: serverKey,
				certresources.CACert:    caCert,
			},
			want: "server cert missing",
		},
		{
			name:       "Invalid certs",
			secretName: testsecrets,
			secretData: map[string][]byte{},
			want:       "server key missing",
		},
		{
			name:       "Valid certs",
			secretName: testsecrets,
			secretData: map[string][]byte{
				certresources.ServerKey:  serverKey,
				certresources.ServerCert: serverCert,
				certresources.CACert:     caCert,
			},
			want: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(interceptorTLSSecretKey, tc.secretName)
			if err := clientSet.Secrets(namespace).Delete(ctx, tc.secretName, metav1.DeleteOptions{}); err != nil && !apiErrors.IsNotFound(err) {
				t.Error(err)
			}
			localObject := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tc.secretName,
					Namespace: namespace,
				},
				Data: tc.secretData,
			}
			// In order to test secret not found scenario
			if os.Getenv(interceptorTLSSecretKey) == testsecrets {
				s, err := clientSet.Secrets(namespace).Create(ctx, localObject, metav1.CreateOptions{})
				if err != nil {
					t.Error(err)
				}
				if err = secretInformer.Informer().GetIndexer().Add(s); err != nil {
					t.Error(err)
				}
			}
			if _, err := GetTLSData(ctx, logger.Sugar()); err != nil && err.Error() != tc.want {
				t.Error(err)
			}
		})
	}
}

func Test_UpdateCACertToClusterInterceptorCRD(t *testing.T) {
	ctx, _ := test.SetupFakeContext(t)
	logger := zaptest.NewLogger(t)
	secretInformer := fakesecretinformer.Get(ctx)
	clientSet := fakekubeclient.Get(ctx).CoreV1()
	t.Setenv(interceptorTLSSecretKey, testsecrets)

	s := createSecretWithData(ctx, t, clientSet)
	if err := secretInformer.Informer().GetIndexer().Add(s); err != nil {
		t.Error(err)
	}

	server, ci := registerAndGetCI(ctx, t, "firstci1", logger)
	if _, err := faketriggersclient.Get(ctx).TriggersV1alpha1().ClusterInterceptors().Create(ctx, &ci, metav1.CreateOptions{}); err != nil {
		t.Error(err)
	}

	UpdateCACertToClusterInterceptorCRD(ctx, server, faketriggersclient.Get(ctx).TriggersV1alpha1(), logger.Sugar(), time.Second)

	time.Sleep(10 * time.Second)
	ciNew, err := faketriggersclient.Get(ctx).TriggersV1alpha1().ClusterInterceptors().Get(ctx, "firstci1", metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}

	if string(ciNew.Spec.ClientConfig.CaBundle) == "" {
		t.Error("caBundle should exist after successful clusterinterceptor creation")
	}
}

func getCerts(ctx context.Context, t *testing.T) ([]byte, []byte, []byte, string) {
	t.Setenv(interceptorTLSSvcKey, testsvc)
	t.Setenv("SYSTEM_NAMESPACE", testns)
	namespace := system.Namespace()

	serverKey, serverCert, caCert, err := certresources.CreateCerts(ctx, os.Getenv(interceptorTLSSvcKey), namespace, time.Now().Add(second))
	if err != nil {
		t.Error(err)
	}
	return serverKey, serverCert, caCert, namespace
}

func registerAndGetCI(ctx context.Context, t *testing.T, ciName string, logger *zap.Logger) (*Server, v1alpha1.ClusterInterceptor) {
	server, err := NewWithCoreInterceptors(interceptors.DefaultSecretGetter(fakekubeclient.Get(ctx).CoreV1()), logger.Sugar())
	if err != nil {
		t.Fatalf("error initializing core interceptors: %v", err)
	}
	server.RegisterInterceptor(ciName, fakeInterceptor{})
	ci := v1alpha1.ClusterInterceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name: ciName,
		},
		Spec: v1alpha1.ClusterInterceptorSpec{
			ClientConfig: v1alpha1.ClientConfig{
				CaBundle: nil,
				URL:      nil,
				Service:  nil,
			},
		},
	}
	return server, ci
}

func createSecretWithData(ctx context.Context, t *testing.T, clientSet v1.CoreV1Interface) *corev1.Secret {
	serverKey, serverCert, caCert, namespace := getCerts(ctx, t)
	if len(serverCert) == 0 && len(caCert) == 0 && len(serverKey) == 0 {
		t.Error("expected serverCert, caCert and serverKey to be created")
	}
	localObject := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      os.Getenv(interceptorTLSSecretKey),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			certresources.ServerKey:  serverKey,
			certresources.ServerCert: serverCert,
			certresources.CACert:     caCert,
		},
	}
	s, err := clientSet.Secrets(namespace).Create(ctx, localObject, metav1.CreateOptions{})
	if err != nil {
		t.Error(err)
	}
	return s
}
