package test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/server"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

// SetupInterceptors creates a httptest server with all coreInterceptors and any passed in webhook interceptor
// It returns a http.Client that can be used to talk to these interceptors
func SetupInterceptors(t *testing.T, k kubernetes.Interface, l *zap.SugaredLogger, webhookInterceptor http.Handler) *http.Client {
	t.Helper()
	// Setup a handler for core interceptors using httptest
	coreInterceptors, err := server.NewWithCoreInterceptors(k, l)
	if err != nil {
		t.Fatalf("failed to initialize core interceptors: %v", err)
	}
	rtr := mux.NewRouter()
	// server core interceptors by matching on req host
	rtr.MatcherFunc(func(r *http.Request, _ *mux.RouteMatch) bool {
		return strings.Contains(r.Host, interceptors.CoreInterceptorsHost)
	}).Handler(coreInterceptors)

	if webhookInterceptor != nil {
		rtr.Handle("/", webhookInterceptor)
	}
	srv := httptest.NewServer(rtr)
	t.Cleanup(func() {
		srv.Close()
	})
	httpClient := srv.Client()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("testServer() url parse err: %v", err)
	}
	httpClient.Transport = &http.Transport{
		Proxy: http.ProxyURL(u),
	}
	return httpClient
}
