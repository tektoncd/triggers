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
	"os"
	"time"

	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/server"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	secretInformer "knative.dev/pkg/client/injection/kube/informers/core/v1/secret"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/system"
	certresources "knative.dev/pkg/webhook/certificates/resources"
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

	serverCert, caCert, err := server.CreateCerts(ctx, kubeclient.Get(ctx).CoreV1(), time.Now().Add(server.Decade), logger)
	if err != nil {
		return
	}

	if err := service.ListAndUpdateClusterInterceptorCRD(ctx, tc.TriggersV1alpha1(), caCert); err != nil {
		return
	}

	// After creating certificates using CreateCerts lets validate validity of created certificates
	service.CheckCertValidity(ctx, serverCert, caCert, kubeclient.Get(ctx).CoreV1(), logger, tc.TriggersV1alpha1(), time.Minute)

	interceptorSecretName := os.Getenv(server.InterceptorTLSSecretKey)
	ticker := time.NewTicker(time.Minute)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				secret, err := secretInformer.Get(ctx).Lister().Secrets(system.Namespace()).Get(interceptorSecretName)
				if err != nil {
					logger.Errorf("failed to fetch secret %v", err)
					return
				}
				caCert, ok := secret.Data[certresources.CACert]
				if !ok {
					logger.Warn("CACert key missing")
					return
				}
				if err := service.ListAndUpdateClusterInterceptorCRD(ctx, tc.TriggersV1alpha1(), caCert); err != nil {
					return
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	tlsData := &tls.Config{
		MinVersion: tls.VersionTLS13,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			secret, err := secretInformer.Get(ctx).Lister().Secrets(system.Namespace()).Get(interceptorSecretName)
			if err != nil {
				logger.Errorf("failed to fetch secret %v", err)
				return nil, nil
			}

			serverKey, ok := secret.Data[certresources.ServerKey]
			if !ok {
				logger.Warn("server key missing")
				return nil, nil
			}
			serverCert, ok := secret.Data[certresources.ServerCert]
			if !ok {
				logger.Warn("server cert missing")
				return nil, nil
			}
			cert, err := tls.X509KeyPair(serverCert, serverKey)
			if err != nil {
				return nil, err
			}
			return &cert, nil
		},
	}
	if err := startServer(ctx, ctx.Done(), mux, tlsData, logger); err != nil {
		logger.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func startServer(ctx context.Context, stop <-chan struct{}, mux *http.ServeMux, tlsData *tls.Config, logger *zap.SugaredLogger) error {

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
		TLSConfig:         tlsData,
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
