/*
Copyright 2019 The Tekton Authors

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
	"net/http"
	"time"

	"go.uber.org/zap"

	dynamicClientset "github.com/tektoncd/triggers/pkg/client/dynamic/clientset"
	"github.com/tektoncd/triggers/pkg/client/dynamic/clientset/tekton"
	"github.com/tektoncd/triggers/pkg/client/informers/externalversions"
	triggerLogging "github.com/tektoncd/triggers/pkg/logging"
	"github.com/tektoncd/triggers/pkg/sink"
	"github.com/tektoncd/triggers/pkg/system"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	cminformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/metrics"
	"knative.dev/pkg/profiling"
	"knative.dev/pkg/signals"
)

const (
	// EventListenerLogKey is the name of the logger for the eventlistener cmd
	EventListenerLogKey = "eventlistener"
	// ConfigName is the name of the ConfigMap that the logging config will be stored in
	ConfigName = "config-logging-triggers"
)

func main() {
	ctx := signals.NewContext()

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to get in cluster config: %v", err)
	}
	ctx, _ = injection.EnableInjectionOrDie(ctx, clusterConfig)

	kubeClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		log.Fatalf("Failed to get the Kubernetes client set: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(clusterConfig)
	if err != nil {
		log.Fatalf("Failed to get the dynamic client: %v", err)
	}
	dynamicCS := dynamicClientset.New(tekton.WithClient(dynamicClient))

	// Watch the logging config map and dynamically update logging levels.
	configMapWatcher := cminformer.NewInformedWatcher(kubeClient, system.GetNamespace())

	logger := triggerLogging.ConfigureLogging(EventListenerLogKey, ConfigName, ctx.Done(), configMapWatcher)
	profilingHandler := profiling.NewHandler(logger, false)
	profilingServer := profiling.NewServer(profilingHandler)
	metrics.MemStatsOrDie(ctx)

	sharedmain.WatchObservabilityConfigOrDie(ctx, configMapWatcher, profilingHandler, logger, EventListenerLogKey)
	logger.Info("Starting configuration manager...")
	if err := configMapWatcher.Start(ctx.Done()); err != nil {
		logger.Fatalw("Failed to start configuration manager", zap.Error(err))
	}
	ctx = logging.WithLogger(ctx, logger)
	defer func() {
		err := logger.Sync()
		if err != nil {
			logger.Fatalf("Failed to sync the logger", zap.Error(err))
		}
	}()

	logger.Info("EventListener pod started")

	sinkArgs, err := sink.GetArgs()
	if err != nil {
		logger.Fatal(err)
	}

	sinkClients, err := sink.ConfigureClients(clusterConfig)
	if err != nil {
		logger.Fatal(err)
	}

	factory := externalversions.NewSharedInformerFactoryWithOptions(sinkClients.TriggersClient,
		30*time.Second, externalversions.WithNamespace(sinkArgs.ElNamespace))
	if sinkArgs.IsMultiNS {
		factory = externalversions.NewSharedInformerFactory(sinkClients.TriggersClient,
			30*time.Second)
	}

	go func(ctx context.Context) {
		factory.Start(ctx.Done())
		<-ctx.Done()
	}(ctx)
	recorder, err := sink.NewRecorder()
	if err != nil {
		logger.Fatal(err)
	}

	// Create EventListener Sink
	r := sink.Sink{
		KubeClientSet:               kubeClient,
		DiscoveryClient:             sinkClients.DiscoveryClient,
		DynamicClient:               dynamicCS,
		TriggersClient:              sinkClients.TriggersClient,
		HTTPClient:                  http.DefaultClient,
		EventListenerName:           sinkArgs.ElName,
		EventListenerNamespace:      sinkArgs.ElNamespace,
		Logger:                      logger,
		Recorder:                    recorder,
		Auth:                        sink.DefaultAuthOverride{},
		EventListenerLister:         factory.Triggers().V1alpha1().EventListeners().Lister(),
		TriggerLister:               factory.Triggers().V1alpha1().Triggers().Lister(),
		TriggerBindingLister:        factory.Triggers().V1alpha1().TriggerBindings().Lister(),
		ClusterTriggerBindingLister: factory.Triggers().V1alpha1().ClusterTriggerBindings().Lister(),
		TriggerTemplateLister:       factory.Triggers().V1alpha1().TriggerTemplates().Lister(),
	}

	// Listen and serve
	logger.Infof("Listen and serve on port %s", sinkArgs.Port)
	mux := http.NewServeMux()

	chain := sink.MiddlewareHandlerFunc(r.HandleEvent).Intercept(sink.NewMetricsRecorderInterceptor(r))
	mux.HandleFunc("/", http.HandlerFunc(chain))

	// For handling Liveness Probe
	// TODO(dibyom): Livness, metrics etc. should be on a separate port
	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "ok")
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", sinkArgs.Port),
		ReadTimeout:  sinkArgs.ELReadTimeOut * time.Second,
		WriteTimeout: sinkArgs.ELWriteTimeOut * time.Second,
		IdleTimeout:  sinkArgs.ELIdleTimeOut * time.Second,
		Handler: http.TimeoutHandler(mux,
			sinkArgs.ELTimeOutHandler*time.Second, "EventListener Timeout!\n"),
	}

	if sinkArgs.Cert == "" && sinkArgs.Key == "" {
		if err := srv.ListenAndServe(); err != nil {
			logger.Fatalf("failed to start eventlistener sink: %v", err)
		}
	} else {
		if err := srv.ListenAndServeTLS(sinkArgs.Cert, sinkArgs.Key); err != nil {
			logger.Fatalf("failed to start eventlistener sink: %v", err)
		}
	}

	err = profilingServer.Shutdown(context.Background())
	if err != nil {
		logger.Fatalf("failed to shutdown profiling server: %v", err)
	}
}
