package server_test

import (
	"context"
	"os"
	"testing"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	faketriggersclient "github.com/tektoncd/triggers/pkg/client/injection/client/fake"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/server"
	"github.com/tektoncd/triggers/test"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
)

type fakeInterceptor struct{}

func (i fakeInterceptor) Process(ctx context.Context, r *v1beta1.InterceptorRequest) *v1beta1.InterceptorResponse {
	return nil
}

func createSecret(t *testing.T) (string, string, []byte, error) {
	if err := os.Setenv("INTERCEPTOR_TLS_SVC_NAME", "testsvc"); err != nil {
		return "", "", []byte{}, err
	}
	if err := os.Setenv("INTERCEPTOR_TLS_SECRET_NAME", "testsecrets"); err != nil {
		return "", "", []byte{}, err
	}
	if err := os.Setenv("SYSTEM_NAMESPACE", "testns"); err != nil {
		return "", "", []byte{}, err
	}
	interceptorSecretName := os.Getenv("INTERCEPTOR_TLS_SECRET_NAME")
	namespace := os.Getenv("SYSTEM_NAMESPACE")

	logger := zaptest.NewLogger(t)
	ctx, _ := test.SetupFakeContext(t)
	clientSet := fakekubeclient.Get(ctx).CoreV1()
	localObject := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      interceptorSecretName,
			Namespace: namespace,
		},
	}
	if _, err := clientSet.Secrets(namespace).Create(ctx, localObject, metav1.CreateOptions{}); err != nil {
		t.Error(err)
	}
	key, sKey, crt, err := server.CreateCerts(ctx, clientSet, logger.Sugar())
	return key, sKey, crt, err
}

func Test_CreateSecret(t *testing.T) {
	key, sKey, crt, err := createSecret(t)
	if err != nil {
		t.Error(err)
	}
	if key == "" && sKey == "" && len(crt) == 0 {
		t.Error("expected key, server and crt to be created")
	}
}

func Test_UpdateCRDWithCaCert(t *testing.T) {
	ctx, _ := test.SetupFakeContext(t)
	logger := zaptest.NewLogger(t)
	key, sKey, crt, err := createSecret(t)
	if err != nil {
		t.Error(err)
	}
	if key == "" && sKey == "" && len(crt) == 0 {
		t.Error("expected key, server and crt to be created")
	}
	server, err := server.NewWithCoreInterceptors(interceptors.DefaultSecretGetter(fakekubeclient.Get(ctx).CoreV1()), logger.Sugar())
	if err != nil {
		t.Fatalf("error initializing core interceptors: %v", err)
	}
	server.RegisterInterceptor("firstci", fakeInterceptor{})
	ci := &v1alpha1.ClusterInterceptor{
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
	var ciList []*v1alpha1.ClusterInterceptor
	ciList = append(ciList, ci)
	if _, err := faketriggersclient.Get(ctx).TriggersV1alpha1().ClusterInterceptors().Create(ctx, ci, metav1.CreateOptions{}); err != nil {
		t.Error(err)
	}

	if err := server.UpdateCRDWithCaCert(ctx, faketriggersclient.Get(ctx).TriggersV1alpha1(), ciList, crt); err != nil {
		t.Error(err)
	}
}
