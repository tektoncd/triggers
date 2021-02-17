/*
Copyright 2021 The Tekton Authors

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

package interceptortype

import (
	"context"

	"knative.dev/pkg/logging"

	interceptortypeinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/interceptortype"
	interceptortypereconciler "github.com/tektoncd/triggers/pkg/client/injection/reconciler/triggers/v1alpha1/interceptortype"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
)

// NewController creates a new instance of an InterceptorType controller.
func NewController() func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)
		interceptorTypeInformer := interceptortypeinformer.Get(ctx)
		reconciler := &Reconciler{}

		impl := interceptortypereconciler.NewImpl(ctx, reconciler, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: ControllerName,
			}
		})

		logger.Info("Setting up event handlers")
		interceptorTypeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.Enqueue,
			UpdateFunc: controller.PassNew(impl.Enqueue),
			DeleteFunc: impl.Enqueue,
		})

		return impl
	}
}
