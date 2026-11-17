/*
Copyright 2026 The Tekton Authors

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
	"os"

	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/cronrunner"
	stresources "github.com/tektoncd/triggers/pkg/reconciler/scheduledtemplate/resources"
	"go.uber.org/zap"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/signals"
)

func main() {
	os.Exit(run())
}

func run() int {
	flag.Parse()

	namespace := os.Getenv(stresources.EnvScheduledTemplateNamespace)
	name := os.Getenv(stresources.EnvScheduledTemplateName)
	if namespace == "" || name == "" {
		log.Printf("%s and %s environment variables are required",
			stresources.EnvScheduledTemplateNamespace, stresources.EnvScheduledTemplateName)
		return 1
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("failed to get in-cluster config: %v", err)
		return 1
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Printf("failed to create kubernetes client: %v", err)
		return 1
	}
	triggersClient, err := triggersclientset.NewForConfig(cfg)
	if err != nil {
		log.Printf("failed to create triggers client: %v", err)
		return 1
	}
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		log.Printf("failed to create dynamic client: %v", err)
		return 1
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Printf("failed to create logger: %v", err)
		return 1
	}
	defer func() { _ = logger.Sync() }()
	sugar := logger.Sugar()

	ctx := signals.NewContext()
	runner := &cronrunner.Runner{
		TriggersClient:  triggersClient,
		DynamicClient:   dynamicClient,
		DiscoveryClient: kubeClient.Discovery(),
		Logger:          sugar,
	}

	if err := runner.Run(ctx, namespace, name); err != nil {
		sugar.Errorw("cronrunner failed", zap.Error(err))
		return 1
	}
	return 0
}
