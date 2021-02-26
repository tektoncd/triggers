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

package logging

import (
	"flag"
	"log"

	"go.uber.org/zap"
	"knative.dev/pkg/configmap"
	cminformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/logging/logkey"
)

// Configure logging
func ConfigureLogging(logKeyString, configName string, stopCh <-chan struct{}, cmw *cminformer.InformedWatcher) *zap.SugaredLogger {
	flag.Parse()
	cm, err := configmap.Load("/etc/config-logging")
	if err != nil {
		log.Fatalf("Error loading logging configuration: %v", err)
	}
	config, err := logging.NewConfigFromMap(cm)
	if err != nil {
		log.Fatalf("Error parsing logging configuration: %v", err)
	}
	logger, atomicLevel := logging.NewLoggerFromConfig(config, logKeyString)

	logger = logger.With(zap.String(logkey.ControllerType, logKeyString))

	logger.Infof("Starting the Configuration %v", logKeyString)

	cmw.Watch(configName, logging.UpdateLevelFromConfigMap(logger, atomicLevel, logKeyString))
	return logger
}
