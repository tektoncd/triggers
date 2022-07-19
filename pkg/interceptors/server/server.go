package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/client/clientset/versioned/typed/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/bitbucket"
	"github.com/tektoncd/triggers/pkg/interceptors/cel"
	"github.com/tektoncd/triggers/pkg/interceptors/github"
	"github.com/tektoncd/triggers/pkg/interceptors/gitlab"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"
)

// Default values for options
const (
	// DefaultHTTPSPort is the port where interceptor service listens on
	defaultHTTPSPort    = 8443
	defaultReadTimeout  = 5 * time.Second
	defaultWriteTimeout = 20 * time.Second
	defaultIdleTimeout  = 60 * time.Second
)

type Options struct {
	// ServiceName is the name of the interceptor service
	// Usually, this will be passed in via a environment variable
	ServiceName string

	// SecretName is the name of k8s secret that contains the interceptor
	// server key/cert and corresponding CA cert that signed them. The
	// server key/cert are used to serve the interceptor and the CA cert
	// is provided to eventListener.
	// If no SecretName is provided, then the webhook serves without TLS.
	SecretName string

	// Port where the interceptor is served.
	Port int

	// ReadTimeout is the http.Server.ReadTimeout for the interceptor server
	ReadTimeout time.Duration

	// WriteTimeout is the http.Server.WriteTimeout for the interceptor server
	WriteTimeout time.Duration

	// IdleTimeout is the http.Server.IdleTimeout for the interceptor server
	IdleTimeout time.Duration

	// Should Namespace come from options as well?

	// TODO: Options for stats and metrics and graceful shutdown
}

func (o *Options) SetDefaults() {
	// TODO: ServiceName, SecretName
	if o.Port == 0 {
		o.Port = defaultHTTPSPort
	}
	if o.ReadTimeout == 0 {
		o.ReadTimeout = defaultReadTimeout
	}
	if o.WriteTimeout == 0 {
		o.WriteTimeout = defaultWriteTimeout
	}
	if o.IdleTimeout == 0 {
		o.IdleTimeout = defaultIdleTimeout
	}

}

type Server struct {
	Options      Options
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

func Main(ctx context.Context, component string, cfg *rest.Config, options Options, interceptors map[string]triggersv1.InterceptorInterface) {
	// set up signals so we handle the first shutdown signal gracefully
	// TODO: Decide if these should be inputs or not
	// ctx := signals.NewContext()
	// cfg := injection.ParseAndGetRESTConfigOrDie()

	// TODO
	// metrics.MemStatsOrDie(ctx)

	ctx, startInformer := injection.EnableInjectionOrDie(ctx, cfg)

	logger, atomicLevel := sharedmain.SetupLoggerOrDie(ctx, component)
	ctx = logging.WithLogger(ctx, logger)
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Fatalf("failed to sync the logger: %s", err)
		}
	}()
	ctx = logging.WithLogger(ctx, logger)
	// Override client-go's warning handler to give us nicely printed warnings.
	rest.SetDefaultWarningHandler(&logging.WarningHandler{Logger: logger})

	cmw := sharedmain.SetupConfigMapWatchOrDie(ctx, logger)
	sharedmain.WatchLoggingConfigOrDie(ctx, cmw, logger, atomicLevel, component)

	kubeClient := kubeclient.Get(ctx)
	// TODO: NewKubeClientSecretGetter should be injected
	startInformer()

	// Set default values before initializing Server
	options.SetDefaults()
	server := &Server{
		Logger:  logger,
		Options: options,
	}
	for path, interceptor := range interceptors {
		server.RegisterInterceptor(path, interceptor)
	}

	mux := http.NewServeMux()
	mux.Handle("/", server)
	mux.HandleFunc("/ready", readyHandler)

	keyFile, certFile, caCert, err := CreateCerts(ctx, kubeClient.CoreV1(), logger)
	if err != nil {
		return
	}

	tc, err := triggersclientset.NewForConfig(cfg)
	if err != nil {
		return
	}

	if err := listAndUpdateClusterInterceptorCRD(ctx, tc, server, caCert); err != nil {
		return
	}
	ticker := time.NewTicker(time.Minute)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := listAndUpdateClusterInterceptorCRD(ctx, tc, server, caCert); err != nil {
					return
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", server.Options.Port),
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		ReadTimeout:  server.Options.ReadTimeout,
		WriteTimeout: server.Options.WriteTimeout,
		IdleTimeout:  server.Options.IdleTimeout,
		Handler:      mux,
	}
	// TODO: User srv.TLSConfig.Certificates or srv.TLSConfig.GetCertificate so we don't have to write to a file
	logger.Infof("Listen and serve on port %d", server.Options.Port)
	if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil {
		logger.Fatalf("failed to start interceptors service: %v", err)
	}

}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
