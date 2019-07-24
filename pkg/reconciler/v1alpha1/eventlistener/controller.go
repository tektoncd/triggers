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
	"time"

	"github.com/knative/pkg/configmap"
	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/injection/clients/kubeclient"
	"github.com/knative/pkg/logging"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	triggersclient "github.com/tektoncd/triggers/pkg/client/injection/client"
	eventlistenerinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/eventlistener"
	"github.com/tektoncd/triggers/pkg/reconciler"
	"k8s.io/client-go/tools/cache"
)

const (
	resyncPeriod = 10 * time.Hour
)

func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	logger := logging.FromContext(ctx)
	kubeclientset := kubeclient.Get(ctx)
	pipelineclientset := pipelineclient.Get(ctx)
	triggersclientset := triggersclient.Get(ctx)
	eventListenerInformer := eventlistenerinformer.Get(ctx)

	opt := reconciler.Options{
		KubeClientSet:     kubeclientset,
		PipelineClientSet: pipelineclientset,
		TriggersClientSet: triggersclientset,
		ConfigMapWatcher:  cmw,
		Logger:            logger,
		ResyncPeriod:      resyncPeriod,
	}

	c := &Reconciler{
		Base:                reconciler.NewBase(opt, eventListenerAgentName),
		eventListenerLister: eventListenerInformer.Lister(),
	}
	impl := controller.NewImpl(c, c.Logger, eventListenerControllerName)

	c.Logger.Info("Setting up event handlers")
	eventListenerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
		DeleteFunc: impl.Enqueue,
	})

	return impl
}
