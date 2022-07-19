package server

import (
	"context"
	"io/ioutil"
	"os"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"

	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	clusterinterceptorsinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clusterinterceptor"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	certresources "knative.dev/pkg/webhook/certificates/resources"
)

const (
	keyFile  = "/tmp/server-key.pem"
	certFile = "/tmp/server-cert.pem"

	decade = 100 * 365 * 24 * time.Hour
)

/*
* Cert flow:
* See if cert exists in secret
* If it does, check if its valid
* If either of the above is false, generate a new cert
* If we generated a new cert, update the secret with the cert
* If we generated a new cert, update write the cert to a specific file location
* Return the cert so we can update the CRD
**/

// TODO:
// 1. Use one place for all interceptor server config options i.e. pass via main instead of env vars here
// 2. Split out function between updating secret and writing file
// 3. Do we even need to write to file?
// 4. Just return errors instead of logging from this function and have the calling function log
// 5. Use secretGetter or a lister?

// CreateCert creates a self signed cert if needed and updates the secret with the cert
// It writes the cert and the key to a file and returns the keyFile, the certFile as well as the caCert
func CreateCerts(ctx context.Context, coreV1Interface corev1.CoreV1Interface, logger *zap.SugaredLogger) (string, string, []byte, error) {
	interceptorSvcName := os.Getenv("INTERCEPTOR_TLS_SVC_NAME")
	interceptorSecretName := os.Getenv("INTERCEPTOR_TLS_SECRET_NAME")
	namespace := os.Getenv("SYSTEM_NAMESPACE")

	secret, err := coreV1Interface.Secrets(namespace).Get(ctx, interceptorSecretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// The secret should be created explicitly by a higher-level system
			// that's responsible for install/updates.  We simply populate the
			// secret information.
			logger.Infof("secret %s is missing", interceptorSecretName)
			return "", "", []byte{}, err
		}
		logger.Infof("error accessing certificate secret %q: %v", interceptorSecretName, err)
		return "", "", []byte{}, err
	}

	var (
		serverKey, serverCert, caCert []byte
		createCerts                   bool
	)
	serverKey, haskey := secret.Data[certresources.ServerKey]
	if !haskey {
		logger.Infof("secret %q is missing key %q", secret.Name, certresources.ServerKey)
		createCerts = true
	}
	serverCert, haskey = secret.Data[certresources.ServerCert]
	if !haskey {
		logger.Infof("secret %q is missing key %q", secret.Name, certresources.ServerCert)
		createCerts = true
	}
	caCert, haskey = secret.Data[certresources.CACert]
	if !haskey {
		logger.Infof("secret %q is missing key %q", secret.Name, certresources.CACert)
		createCerts = true
	}

	// TODO: Certification validation and rotation is pending

	if createCerts {
		serverKey, serverCert, caCert, err = certresources.CreateCerts(ctx, interceptorSvcName, namespace, time.Now().Add(decade))
		if err != nil {
			logger.Errorf("failed to create certs : %v", err)
			return "", "", []byte{}, err
		}

		secret.Data = map[string][]byte{
			certresources.ServerKey:  serverKey,
			certresources.ServerCert: serverCert,
			certresources.CACert:     caCert,
		}
		if _, err = coreV1Interface.Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			logger.Errorf("failed to update secret : %v", err)
			return "", "", []byte{}, err
		}
	}

	// write serverKey to file so that it can be passed while running https server.
	if err = ioutil.WriteFile(keyFile, serverKey, 0600); err != nil {
		logger.Errorf("failed to write serverKey file %v", err)
		return "", "", []byte{}, err
	}

	// write serverCert to file so that it can be passed while running https server.
	if err = ioutil.WriteFile(certFile, serverCert, 0600); err != nil {
		logger.Errorf("failed to write serverCert file %v", err)
		return "", "", []byte{}, err
	}
	return keyFile, certFile, caCert, nil
}

func listAndUpdateClusterInterceptorCRD(ctx context.Context, tc *triggersclientset.Clientset, service *Server, caCert []byte) error {
	clusterInterceptorList, err := clusterinterceptorsinformer.Get(ctx).Lister().List(labels.NewSelector())
	if err != nil {
		return err
	}

	if err := service.UpdateCRDWithCaCert(ctx, tc.TriggersV1alpha1(), clusterInterceptorList, caCert); err != nil {
		return err
	}
	return nil
}
