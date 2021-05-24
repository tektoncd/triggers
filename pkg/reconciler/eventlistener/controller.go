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

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersclient "github.com/tektoncd/triggers/pkg/client/injection/client"
	eventlistenerinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/eventlistener"
	eventlistenerreconciler "github.com/tektoncd/triggers/pkg/client/injection/reconciler/triggers/v1alpha1/eventlistener"
	dynamicduck "github.com/tektoncd/triggers/pkg/dynamic"
	"k8s.io/client-go/tools/cache"
	duckinformer "knative.dev/pkg/client/injection/ducks/duck/v1/podspecable"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	deployinformer "knative.dev/pkg/client/injection/kube/informers/apps/v1/deployment"
	configmapinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/configmap"
	serviceinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/service"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/logging"
)

// NewController creates a new instance of an EventListener controller.
func NewController(config Config) func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)
		dynamicclientset := dynamicclient.Get(ctx)
		kubeclientset := kubeclient.Get(ctx)
		triggersclientset := triggersclient.Get(ctx)
		eventListenerInformer := eventlistenerinformer.Get(ctx)
		deploymentInformer := deployinformer.Get(ctx)
		serviceInformer := serviceinformer.Get(ctx)

		reconciler := &Reconciler{
			DynamicClientSet:    dynamicclientset,
			KubeClientSet:       kubeclientset,
			TriggersClientSet:   triggersclientset,
			configmapLister:     configmapinformer.Get(ctx).Lister(),
			deploymentLister:    deploymentInformer.Lister(),
			eventListenerLister: eventListenerInformer.Lister(),
			serviceLister:       serviceInformer.Lister(),

			config: config,
		}

		impl := eventlistenerreconciler.NewImpl(ctx, reconciler, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: ControllerName,
			}
		})

		logger.Info("Setting up event handlers")

		reconciler.podspecableTracker = dynamicduck.NewListableTracker(ctx, duckinformer.Get, impl)

		eventListenerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.Enqueue,
			UpdateFunc: controller.PassNew(impl.Enqueue),
			DeleteFunc: impl.Enqueue,
		})

		deploymentInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: controller.FilterControllerGVK(v1alpha1.SchemeGroupVersion.WithKind("EventListener")),
			Handler:    controller.HandleAll(impl.EnqueueControllerOf),
		})

		serviceInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: controller.FilterControllerGVK(v1alpha1.SchemeGroupVersion.WithKind("EventListener")),
			Handler:    controller.HandleAll(impl.EnqueueControllerOf),
		})

		return impl
	}
}
