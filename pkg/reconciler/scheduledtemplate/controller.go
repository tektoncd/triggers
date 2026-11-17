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

package scheduledtemplate

import (
	"context"

	cfg "github.com/tektoncd/triggers/pkg/apis/config"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	scheduledtemplateinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/scheduledtemplate"
	cronjobinformer "github.com/tektoncd/triggers/pkg/client/injection/kube/informers/batch/v1/cronjob"
	scheduledtemplatereconciler "github.com/tektoncd/triggers/pkg/client/injection/reconciler/triggers/v1alpha1/scheduledtemplate"
	"k8s.io/client-go/tools/cache"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

// NewController creates a new instance of a ScheduledTemplate controller.
func NewController(image string) func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)
		scheduledTemplateInformer := scheduledtemplateinformer.Get(ctx)
		cronJobInformer := cronjobinformer.Get(ctx)

		reconciler := &Reconciler{
			KubeClientSet: kubeclient.Get(ctx),
			Image:         image,
		}

		impl := scheduledtemplatereconciler.NewImpl(ctx, reconciler, func(_ *controller.Impl) controller.Options {
			configStore := cfg.NewStore(logger.Named("config-store"))
			configStore.WatchConfigs(cmw)
			return controller.Options{
				AgentName:   ControllerName,
				ConfigStore: configStore,
			}
		})

		if _, err := scheduledTemplateInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue)); err != nil {
			logging.FromContext(ctx).Panicf("Couldn't register ScheduledTemplate informer event handler: %w", err)
		}

		if _, err := cronJobInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: controller.FilterController(&v1alpha1.ScheduledTemplate{}),
			Handler:    controller.HandleAll(impl.EnqueueControllerOf),
		}); err != nil {
			logging.FromContext(ctx).Panicf("Couldn't register CronJob informer event handler: %w", err)
		}

		return impl
	}
}
