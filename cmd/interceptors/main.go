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
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	clusterinterceptorsinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clusterinterceptor"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/server"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"
	kubeclientset "k8s.io/client-go/kubernetes"
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

	kubeClient, err := kubeclientset.NewForConfig(cfg)
	if err != nil {
		logger.Errorf("failed to create new Clientset for the given config: %v", err)
		return
	}

	service, err := server.NewWithCoreInterceptors(interceptors.DefaultSecretGetter(kubeclient.Get(ctx).CoreV1()), logger)
	if err != nil {
		logger.Errorf("failed to initialize core interceptors: %s", err)
		return
	}
	startInformer()

	mux := http.NewServeMux()
	mux.Handle("/", service)
	mux.HandleFunc("/ready", handler)

	keyFile, certFile, caCert, err := server.CreateCerts(ctx, kubeClient.CoreV1(), logger)
	if err != nil {
		return
	}

	tc, err := triggersclientset.NewForConfig(cfg)
	if err != nil {
		return
	}

	if err := listAndUpdateClusterInterceptorCRD(ctx, tc, service, caCert); err != nil {
		return
	}
	ticker := time.NewTicker(time.Minute)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := listAndUpdateClusterInterceptorCRD(ctx, tc, service, caCert); err != nil {
					return
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", HTTPSPort),
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		Handler:      mux,
	}
	logger.Infof("Listen and serve on port %d", HTTPSPort)
	if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil {
		logger.Fatalf("failed to start interceptors service: %v", err)
	}

}

func handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func listAndUpdateClusterInterceptorCRD(ctx context.Context, tc *triggersclientset.Clientset, service *server.Server, caCert []byte) error {
	clusterInterceptorList, err := clusterinterceptorsinformer.Get(ctx).Lister().List(labels.NewSelector())
	if err != nil {
		return err
	}

	if err := service.UpdateCRDWithCaCert(ctx, tc.TriggersV1alpha1(), clusterInterceptorList, caCert); err != nil {
		return err
	}
	return nil
}
