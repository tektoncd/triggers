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

package dynamic

import (
	"context"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/apis/duck"
	"knative.dev/pkg/controller"
)

type ListableTracker interface {
	WatchOnDynamicObject(ctx context.Context, gvr schema.GroupVersionResource) error
}

type listableTracker struct {
	informerFactory duck.InformerFactory
	impl            *controller.Impl
}

// NewListableTracker creates a new ListableTracker, backed by a TypedInformerFactory.
func NewListableTracker(ctx context.Context, getter func(ctx context.Context) duck.InformerFactory, impl *controller.Impl) ListableTracker {
	return &listableTracker{
		informerFactory: getter(ctx),
		impl:            impl,
	}
}

func (t *listableTracker) WatchOnDynamicObject(ctx context.Context, gvr schema.GroupVersionResource) error {
	return t.watchOnDynamicObject(ctx, gvr)
}

func (t *listableTracker) watchOnDynamicObject(ctx context.Context, gvr schema.GroupVersionResource) error {
	shInformer, _, err := t.informerFactory.Get(ctx, gvr)
	if err != nil {
		return err
	}
	_, err = shInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterController(&v1beta1.EventListener{}),
		Handler:    controller.HandleAll(t.impl.EnqueueControllerOf),
	})
	return err
}
