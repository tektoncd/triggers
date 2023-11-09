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

package eventlistener

import (
	"context"

	cfg "github.com/tektoncd/triggers/pkg/apis/config"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggersclient "github.com/tektoncd/triggers/pkg/client/injection/client"
	eventlistenerinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/eventlistener"
	eventlistenerreconciler "github.com/tektoncd/triggers/pkg/client/injection/reconciler/triggers/v1beta1/eventlistener"
	dynamicduck "github.com/tektoncd/triggers/pkg/dynamic"
	"github.com/tektoncd/triggers/pkg/reconciler/eventlistener/resources"
	"github.com/tektoncd/triggers/pkg/reconciler/metrics"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	reconcilersource "knative.dev/eventing/pkg/reconciler/source"
	duckinformer "knative.dev/pkg/client/injection/ducks/duck/v1/podspecable"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	filtereddeployinformer "knative.dev/pkg/client/injection/kube/informers/apps/v1/deployment/filtered"
	filteredserviceinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/service/filtered"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/logging"
)

// NewController creates a new instance of an EventListener controller.
func NewController(config resources.Config) func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)
		dynamicclientset := dynamicclient.Get(ctx)
		kubeclientset := kubeclient.Get(ctx)
		triggersclientset := triggersclient.Get(ctx)
		eventListenerInformer := eventlistenerinformer.Get(ctx)
		deploymentInformer := filtereddeployinformer.Get(ctx, labels.FormatLabels(resources.DefaultStaticResourceLabels))
		serviceInformer := filteredserviceinformer.Get(ctx, labels.FormatLabels(resources.DefaultStaticResourceLabels))

		reconciler := &Reconciler{
			DynamicClientSet:  dynamicclientset,
			KubeClientSet:     kubeclientset,
			TriggersClientSet: triggersclientset,
			deploymentLister:  deploymentInformer.Lister(),
			serviceLister:     serviceInformer.Lister(),
			configAcc:         reconcilersource.WatchConfigurations(ctx, "eventlistener", cmw),
			config:            config,
			Metrics:           metrics.Get(ctx),
		}

		impl := eventlistenerreconciler.NewImpl(ctx, reconciler, func(impl *controller.Impl) controller.Options {
			configStore := cfg.NewStore(logger.Named("config-store"))
			configStore.WatchConfigs(cmw)
			return controller.Options{
				AgentName:   ControllerName,
				ConfigStore: configStore,
			}
		})

		reconciler.podspecableTracker = dynamicduck.NewListableTracker(ctx, duckinformer.Get, impl)

		if _, err := eventListenerInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue)); err != nil {
			logging.FromContext(ctx).Panicf("Couldn't register EventListener informer event handler: %w", err)
		}

		if _, err := deploymentInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: controller.FilterController(&v1beta1.EventListener{}),
			Handler:    controller.HandleAll(impl.EnqueueControllerOf),
		}); err != nil {
			logging.FromContext(ctx).Panicf("Couldn't register Deployment informer event handler: %w", err)
		}

		if _, err := serviceInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: controller.FilterController(&v1beta1.EventListener{}),
			Handler:    controller.HandleAll(impl.EnqueueControllerOf),
		}); err != nil {
			logging.FromContext(ctx).Panicf("Couldn't register Service informer event handler: %w", err)
		}

		return impl
	}
}
