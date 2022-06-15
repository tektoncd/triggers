/*
Copyright 2022 The Tekton Authors

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

package namespacedinterceptor

import (
	"context"

	namespacedinterceptorinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/namespacedinterceptor"
	namespacedinterceptorreconciler "github.com/tektoncd/triggers/pkg/client/injection/reconciler/triggers/v1alpha1/namespacedinterceptor"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
)

func NewController() func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		namespacedInterceptorInformer := namespacedinterceptorinformer.Get(ctx)
		reconciler := &Reconciler{}

		impl := namespacedinterceptorreconciler.NewImpl(ctx, reconciler, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: ControllerName,
			}
		})

		namespacedInterceptorInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

		return impl
	}
}
