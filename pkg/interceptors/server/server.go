package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tektoncd/triggers/pkg/interceptors/bitbucket"
	"github.com/tektoncd/triggers/pkg/interceptors/cel"
	"github.com/tektoncd/triggers/pkg/interceptors/github"
	"github.com/tektoncd/triggers/pkg/interceptors/gitlab"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

type Server struct {
	KubeClient   kubernetes.Interface
	Logger       *zap.SugaredLogger
	interceptors map[string]v1alpha1.InterceptorInterface
}

func NewWithCoreInterceptors(k kubernetes.Interface, l *zap.SugaredLogger) (*Server, error) {
	i := map[string]v1alpha1.InterceptorInterface{
		"bitbucket": bitbucket.NewInterceptor(k, l),
		"cel":       cel.NewInterceptor(k, l),
		"github":    github.NewInterceptor(k, l),
		"gitlab":    gitlab.NewInterceptor(k, l),
	}

	for k, v := range i {
		if v == nil {
			return nil, fmt.Errorf("interceptor %s failed to initialize", k)
		}
	}
	s := Server{
		KubeClient:   k,
		Logger:       l,
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
	var ii v1alpha1.InterceptorInterface

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
	var ireq v1alpha1.InterceptorRequest
	if err := json.Unmarshal(body.Bytes(), &ireq); err != nil {
		return nil, badRequest(fmt.Errorf("failed to parse body as InterceptorRequest: %w", err))
	}
	is.Logger.Debugf("Interceptor Request is: %+v", ireq)
	iresp := ii.Process(ctx, &ireq)
	respBytes, err := json.Marshal(iresp)
	if err != nil {
		return nil, internal(err)
	}
	return respBytes, nil
}
