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
	"log"

	"github.com/knative/pkg/configmap"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"
	"github.com/knative/pkg/signals"
	"github.com/knative/pkg/webhook"
	"github.com/tektoncd/pipeline/pkg/system"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// WebhookLogKey is the name of the logger for the webhook cmd
const (
	WebhookLogKey = "webhook"
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
	logger, atomicLevel := logging.NewLoggerFromConfig(config, WebhookLogKey)

	defer func() {
		err := logger.Sync()
		if err != nil {
			logger.Fatal("Failed to sync the logger", zap.Error(err))
		}
	}()

	logger = logger.With(zap.String(logkey.ControllerType, "webhook"))

	logger.Info("Starting the Configuration Webhook")

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Fatal("Failed to get in cluster config", zap.Error(err))
	}

	kubeClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		logger.Fatal("Failed to get the client set", zap.Error(err))
	}
	// Watch the logging config map and dynamically update logging levels.
	configMapWatcher := configmap.NewInformedWatcher(kubeClient, system.GetNamespace())
	configMapWatcher.Watch(ConfigName, logging.UpdateLevelFromConfigMap(logger, atomicLevel, WebhookLogKey))
	if err = configMapWatcher.Start(stopCh); err != nil {
		logger.Fatalf("failed to start configuration manager: %v", err)
	}

	options := webhook.ControllerOptions{
		ServiceName:    "tekton-triggers-webhook",
		DeploymentName: "tekton-triggers-webhook",
		Namespace:      system.GetNamespace(),
		Port:           8443,
		SecretName:     "triggers-webhook-certs",
		WebhookName:    "triggers-webhook.tekton.dev",
	}
	//TODO add validations here
	controller := webhook.AdmissionController{
		Client:  kubeClient,
		Options: options,
		Handlers: map[schema.GroupVersionKind]webhook.GenericCRD{
			v1alpha1.SchemeGroupVersion.WithKind("EventListener"):   &v1alpha1.EventListener{},
			v1alpha1.SchemeGroupVersion.WithKind("TriggerBinding"):  &v1alpha1.TriggerBinding{},
			v1alpha1.SchemeGroupVersion.WithKind("TriggerTemplate"): &v1alpha1.TriggerTemplate{},
		},
		Logger: logger,
		DisallowUnknownFields: true,
	}

	if err := controller.Run(stopCh); err != nil {
		logger.Fatal("Error running admission controller", zap.Error(err))
	}
}
