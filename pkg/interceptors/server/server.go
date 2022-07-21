package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/client/clientset/versioned/typed/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/bitbucket"
	"github.com/tektoncd/triggers/pkg/interceptors/cel"
	"github.com/tektoncd/triggers/pkg/interceptors/github"
	"github.com/tektoncd/triggers/pkg/interceptors/gitlab"
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

type Server struct {
	Logger       *zap.SugaredLogger
	interceptors map[string]triggersv1.InterceptorInterface
}

// RegisterInterceptor sets up the interceptor to be served at the specfied path
func (is *Server) RegisterInterceptor(path string, interceptor triggersv1.InterceptorInterface) {
	if is.interceptors == nil {
		is.interceptors = map[string]triggersv1.InterceptorInterface{}
	}
	is.interceptors[path] = interceptor
}

func NewWithCoreInterceptors(sg interceptors.SecretGetter, logger *zap.SugaredLogger) (*Server, error) {
	i := map[string]triggersv1.InterceptorInterface{
		"bitbucket": bitbucket.NewInterceptor(sg),
		"cel":       cel.NewInterceptor(sg),
		"github":    github.NewInterceptor(sg),
		"gitlab":    gitlab.NewInterceptor(sg),
	}

	for k, v := range i {
		if v == nil {
			return nil, fmt.Errorf("interceptor %s failed to initialize", k)
		}
	}
	s := Server{
		Logger:       logger,
		interceptors: i,
	}
	return &s, nil
}

func (is *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, err := is.ExecuteInterceptor(r)
	if err != nil {
		switch e := err.(type) {
		case Error:
			is.Logger.Infof("HTTP %d - %s", e.Status(), e)
			http.Error(w, e.Error(), e.Status())
		default:
			is.Logger.Errorf("Non Status Error: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
	w.Header().Add("Content-Type", "application/json")
	if _, err := w.Write(b); err != nil {
		is.Logger.Errorf("failed to write response: %s", err)
	}
}

// Error represents a handler error. It provides methods for a HTTP status
// code and embeds the built-in error interface.
type Error interface {
	error
	Status() int
}

// HTTPError represents an error with an associated HTTP status code.
type HTTPError struct {
	Code int
	Err  error
}

// Allows HTTPError to satisfy the error interface.
func (se HTTPError) Error() string {
	return se.Err.Error()
}

// Returns our HTTP status code.
func (se HTTPError) Status() int {
	return se.Code
}

func badRequest(err error) HTTPError {
	return HTTPError{Code: http.StatusBadRequest, Err: err}
}

func internal(err error) HTTPError {
	return HTTPError{Code: http.StatusInternalServerError, Err: err}
}

func (is *Server) ExecuteInterceptor(r *http.Request) ([]byte, error) {
	var ii triggersv1.InterceptorInterface

	// Find correct interceptor
	ii, ok := is.interceptors[strings.TrimPrefix(strings.ToLower(r.URL.Path), "/")]
	if !ok {
		return nil, badRequest(fmt.Errorf("path did not match any interceptors"))
	}

	// Create a context
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var body bytes.Buffer
	defer r.Body.Close()
	if _, err := io.Copy(&body, r.Body); err != nil {
		return nil, internal(fmt.Errorf("failed to read body: %w", err))
	}
	var ireq triggersv1.InterceptorRequest
	if err := json.Unmarshal(body.Bytes(), &ireq); err != nil {
		return nil, badRequest(fmt.Errorf("failed to parse body as InterceptorRequest: %w", err))
	}
	is.Logger.Debugf("Interceptor Request is: %+v", ireq)
	iresp := ii.Process(ctx, &ireq)
	is.Logger.Infof("Interceptor response is: %+v", iresp)
	respBytes, err := json.Marshal(iresp)
	if err != nil {
		return nil, internal(err)
	}
	return respBytes, nil
}

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

// UpdateCRDWithCaCert updates clusterinterceptor crd caBundle with caCert
func (is *Server) UpdateCRDWithCaCert(ctx context.Context, triggersV1Alpha1 triggersv1alpha1.TriggersV1alpha1Interface,
	ci []*v1alpha1.ClusterInterceptor, caCert []byte) error {
	for i := range ci {
		if _, ok := is.interceptors[ci[i].Name]; ok {
			if bytes.Equal(ci[i].Spec.ClientConfig.CaBundle, []byte{}) {
				ci[i].Spec.ClientConfig.CaBundle = caCert
				if _, err := triggersV1Alpha1.ClusterInterceptors().Update(ctx, ci[i], metav1.UpdateOptions{}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
