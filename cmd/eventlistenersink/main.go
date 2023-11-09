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
	"log"

	"k8s.io/client-go/dynamic"
	kubeclientset "k8s.io/client-go/kubernetes"

	evadapter "knative.dev/eventing/pkg/adapter/v2"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/signals"

	"github.com/tektoncd/triggers/pkg/adapter"
	dynamicClientset "github.com/tektoncd/triggers/pkg/client/dynamic/clientset"
	"github.com/tektoncd/triggers/pkg/client/dynamic/clientset/tekton"
	triggersclient "github.com/tektoncd/triggers/pkg/client/injection/client"
	"github.com/tektoncd/triggers/pkg/sink"
	"github.com/tektoncd/triggers/pkg/sink/cloudevent"
)

const (
	// EventListenerLogKey is the name of the logger for the eventlistener cmd
	EventListenerLogKey = "eventlistener"
)

func main() {
	cfg := injection.ParseAndGetRESTConfigOrDie()

	ctx := signals.NewContext()
	ctx = injection.WithConfig(ctx, cfg)

	dc := dynamic.NewForConfigOrDie(cfg)
	dClientSet := dynamicClientset.New(tekton.WithClient(dc))
	ctx = context.WithValue(ctx, dynamicclient.Key{}, dClientSet)

	sinkArgs, err := sink.GetArgs()
	if err != nil {
		log.Fatal(err.Error())
	}
	recorder, err := sink.NewRecorder()
	if err != nil {
		log.Fatal(err.Error())
	}

	if !sinkArgs.IsMultiNS {
		ctx = injection.WithNamespaceScope(ctx, sinkArgs.ElNamespace)
	}

	ctx, informers := injection.Default.SetupInformers(ctx, cfg)
	if err := controller.StartInformers(ctx.Done(), informers...); err != nil {
		log.Fatal("failed to start informers:", err)
	}

	kubeClient := kubeclient.Get(ctx).(*kubeclientset.Clientset)
	triggersClient := triggersclient.Get(ctx)
	ceClient := cloudevent.Get(ctx)

	sinkClients := sink.Clients{
		DiscoveryClient: kubeClient.Discovery(),
		RESTClient:      kubeClient.RESTClient(),
		TriggersClient:  triggersClient,
		K8sClient:       kubeClient,
		CEClient:        ceClient,
	}

	evadapter.MainWithContext(ctx, EventListenerLogKey, adapter.NewEnvConfig, adapter.New(sinkArgs, sinkClients, recorder))
}
