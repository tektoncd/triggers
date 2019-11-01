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
	"flag"
	"fmt"
	"github.com/tektoncd/pipeline/pkg/system"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/logging/logkey"
	"knative.dev/pkg/signals"
	"log"
	"net/http"

	"github.com/tektoncd/triggers/pkg/sink"
)

// EventListenerLogKey is the name of the logger for the eventlistener cmd
const (
	EventListenerLogKey = "eventlistener"
	// ConfigName is the name of the ConfigMap that the logging config will be stored in
	ConfigName = "config-logging-triggers"
)

func main() {
	flag.Parse()
	cm, err := configmap.Load("/etc/config-logging")
	if err != nil {
		log.Fatalf("Error loading logging configuration: %v", err)
	}
	config, err := logging.NewConfigFromMap(cm)
	if err != nil {
		log.Fatalf("Error parsing logging configuration: %v", err)
	}
	logger, atomicLevel := logging.NewLoggerFromConfig(config, EventListenerLogKey)

	defer func() {
		err := logger.Sync()
		if err != nil {
			logger.Fatalf("Failed to sync the logger", zap.Error(err))
		}
	}()

	logger = logger.With(zap.String(logkey.ControllerType, "eventlistener"))

	logger.Infof("Starting the Configuration EventListener")

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Fatalf("Failed to get in cluster config", zap.Error(err))
	}

	kubeClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		logger.Fatalf("Failed to get the client set", zap.Error(err))
	}

	// Watch the logging config map and dynamically update logging levels.
	configMapWatcher := configmap.NewInformedWatcher(kubeClient, system.GetNamespace())
	configMapWatcher.Watch(ConfigName, logging.UpdateLevelFromConfigMap(logger, atomicLevel, EventListenerLogKey))
	if err = configMapWatcher.Start(stopCh); err != nil {
		logger.Fatalf("failed to start configuration manager: %v", err)
	}

	logger.Info("EventListener pod started")

	sinkArgs, err := sink.GetArgs()
	if err != nil {
		logger.Fatal(err)
	}

	sinkClients, err := sink.ConfigureClients()
	if err != nil {
		logger.Fatal(err)
	}

	// Create sink Resource
	r := sink.Resource{
		DiscoveryClient:        sinkClients.DiscoveryClient,
		RESTClient:             sinkClients.RESTClient,
		TriggersClient:         sinkClients.TriggersClient,
		PipelineClient:         sinkClients.PipelineClient,
		HTTPClient:             http.DefaultClient, // TODO: Use a different client since the default client has weird timeout values
		EventListenerName:      sinkArgs.ElName,
		EventListenerNamespace: sinkArgs.ElNamespace,
		Logger:                 logger,
	}

	// Listen and serve
	logger.Infof("Listen and serve on port %s", sinkArgs.Port)
	http.HandleFunc("/", r.HandleEvent)
	logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", sinkArgs.Port), nil))
}
