package server

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
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
	"knative.dev/pkg/system"
	certresources "knative.dev/pkg/webhook/certificates/resources"
)

const (
	Decade                  = 100 * 365 * 24 * time.Hour
	InterceptorTLSSecretKey = "INTERCEPTOR_TLS_SECRET_NAME"
	InterceptorTLSSvcKey    = "INTERCEPTOR_TLS_SVC_NAME"
)

type keypairReloader struct {
	caCertData     []byte
	serverCertData []byte
}

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

func CreateCerts(ctx context.Context, coreV1Interface corev1.CoreV1Interface, noAfter time.Time, logger *zap.SugaredLogger) ([]byte, []byte, error) {
	interceptorSvcName := os.Getenv(InterceptorTLSSvcKey)
	interceptorSecretName := os.Getenv(InterceptorTLSSecretKey)
	namespace := system.Namespace()

	secret, err := coreV1Interface.Secrets(namespace).Get(ctx, interceptorSecretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// The secret should be created explicitly by a higher-level system
			// that's responsible for install/updates.  We simply populate the
			// secret information.
			logger.Infof("secret %s is missing", interceptorSecretName)
			return []byte{}, []byte{}, err
		}
		logger.Infof("error accessing certificate secret %q: %v", interceptorSecretName, err)
		return []byte{}, []byte{}, err
	}

	serverKey, serverCert, caCert, err := certresources.CreateCerts(ctx, interceptorSvcName, namespace, noAfter)
	if err != nil {
		logger.Errorf("failed to create certs : %v", err)
		return []byte{}, []byte{}, err
	}

	secret.Data = map[string][]byte{
		certresources.ServerKey:  serverKey,
		certresources.ServerCert: serverCert,
		certresources.CACert:     caCert,
	}
	if _, err = coreV1Interface.Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		logger.Errorf("failed to update secret : %v", err)
		return []byte{}, []byte{}, err
	}

	return serverCert, caCert, nil
}

// updateCRDWithCaCert updates clusterinterceptor crd caBundle with caCert
func (is *Server) updateCRDWithCaCert(ctx context.Context, triggersV1Alpha1 triggersv1alpha1.TriggersV1alpha1Interface,
	ci []v1alpha1.ClusterInterceptor, caCert []byte) error {
	for i := range ci {
		if _, ok := is.interceptors[ci[i].Name]; ok {
			if bytes.Equal(ci[i].Spec.ClientConfig.CaBundle, []byte{}) || !bytes.Equal(ci[i].Spec.ClientConfig.CaBundle, caCert) {
				ci[i].Spec.ClientConfig.CaBundle = caCert
				if _, err := triggersV1Alpha1.ClusterInterceptors().Update(ctx, &ci[i], metav1.UpdateOptions{}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (is *Server) CheckCertValidity(ctx context.Context, serverCert, caCert []byte, coreV1Interface corev1.CoreV1Interface,
	logger *zap.SugaredLogger, tc triggersv1alpha1.TriggersV1alpha1Interface, tickerTime time.Duration) {

	result := &keypairReloader{
		caCertData:     caCert,
		serverCertData: serverCert,
	}

	ticker := time.NewTicker(tickerTime)
	quit := make(chan struct{})
	var (
		cert *x509.Certificate
		err  error
	)

	go func() {
		for {
			select {
			case <-ticker.C:
				// Check the expiration date of the certificate to see if it needs to be updated
				roots := x509.NewCertPool()
				ok := roots.AppendCertsFromPEM(result.caCertData)
				if !ok {
					logger.Error("failed to parse root certificate")
				}
				block, _ := pem.Decode(result.serverCertData)
				if block == nil {
					logger.Error("failed to parse certificate PEM")
				} else {
					cert, err = x509.ParseCertificate(block.Bytes)
					if err != nil {
						logger.Errorf("failed to parse certificate: %v", err.Error())
					}
				}

				opts := x509.VerifyOptions{
					Roots: roots,
				}

				if _, err := cert.Verify(opts); err != nil {
					logger.Errorf("failed to verify certificate: %v", err.Error())

					serverCertNew, caCertNew, err := CreateCerts(ctx, coreV1Interface, time.Now().Add(Decade), logger)
					if err != nil {
						logger.Errorf("failed to create certs %v", err)
					}

					result = &keypairReloader{
						caCertData:     caCertNew,
						serverCertData: serverCertNew,
					}
					if err := is.ListAndUpdateClusterInterceptorCRD(ctx, tc, caCertNew); err != nil {
						logger.Error(err.Error())
					}
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (is *Server) ListAndUpdateClusterInterceptorCRD(ctx context.Context, tc triggersv1alpha1.TriggersV1alpha1Interface, caCert []byte) error {
	clusterInterceptorList, err := tc.ClusterInterceptors().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	if err := is.updateCRDWithCaCert(ctx, tc, clusterInterceptorList.Items, caCert); err != nil {
		return err
	}
	return nil
}
