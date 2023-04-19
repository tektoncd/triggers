/*
Copyright 2020 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/server"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
)

const (
	// HTTPSPort is the port where interceptor service listens on
	HTTPSPort    = 8443
	readTimeout  = 5 * time.Second
	writeTimeout = 20 * time.Second
	idleTimeout  = 60 * time.Second
)

func main() {
	// set up signals so we handle the first shutdown signal gracefully
	ctx := signals.NewContext()

	cfg := injection.ParseAndGetRESTConfigOrDie()

	ctx, startInformer := injection.EnableInjectionOrDie(ctx, cfg)

	zap, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize logger: %s", err)
	}
	logger := zap.Sugar()
	ctx = logging.WithLogger(ctx, logger)
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Fatalf("failed to sync the logger: %s", err)
		}
	}()

	service, err := server.NewWithCoreInterceptors(interceptors.DefaultSecretGetter(kubeclient.Get(ctx).CoreV1()), logger)
	if err != nil {
		logger.Errorf("failed to initialize core interceptors: %s", err)
		return
	}
	startInformer()

	mux := http.NewServeMux()
	mux.Handle("/", service)
	mux.HandleFunc("/ready", handler)

	tc, err := triggersclientset.NewForConfig(cfg)
	if err != nil {
		return
	}

	server.CreateAndValidateCerts(ctx, kubeclient.Get(ctx).CoreV1(), logger, service, tc.TriggersV1alpha1())

	// watch for caCert existence in clusterInterceptor, update with new caCert if its missing in clusterInterceptor
	server.UpdateCACertToClusterInterceptorCRD(ctx, service, tc.TriggersV1alpha1(), logger, time.Minute)

	if err := startServer(ctx, ctx.Done(), mux, logger); err != nil {
		logger.Fatal(err)
	}
}

// revive:disable:unused-parameter

func handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func startServer(ctx context.Context, stop <-chan struct{}, mux *http.ServeMux, logger *zap.SugaredLogger) error {

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", HTTPSPort),
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		ReadHeaderTimeout: readTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		Handler:           mux,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
				return server.GetTLSData(ctx, logger)
			},
		},
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		logger.Infof("Listen and serve on port %d", HTTPSPort)
		if err := srv.ListenAndServeTLS("", ""); err != nil {
			logger.Errorf("failed to start interceptors server: %v", err)
			return err
		}
		return nil
	})

	select {
	case <-stop:
		eg.Go(func() error {
			// As we start to shutdown, disable keep-alives to avoid clients hanging onto connections.
			srv.SetKeepAlivesEnabled(false)

			return srv.Shutdown(context.Background())
		})

		// Wait for all outstanding go routined to terminate, including our new one.
		return eg.Wait()

	case <-ctx.Done():
		return fmt.Errorf("interceptors server bootstrap failed %w", ctx.Err())
	}
}
