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

package metrics

import (
	"context"

	ciInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clusterinterceptor"
	ctbInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/clustertriggerbinding"
	elInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/eventlistener"
	tbInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/triggerbinding"
	ttInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/triggertemplate"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
)

func init() {
	injection.Default.RegisterClient(func(ctx context.Context, _ *rest.Config) context.Context { return WithClient(ctx) })
	injection.Default.RegisterInformer(WithInformer)
}

// RecorderKey is used for associating the Recorder inside the context.Context.
type RecorderKey struct{}

func WithClient(ctx context.Context) context.Context {
	rec, err := NewRecorder(ctx)
	if err != nil {
		logging.FromContext(ctx).Errorf("Failed to create trigger metrics recorder %v", err)
	}
	return context.WithValue(ctx, RecorderKey{}, rec)
}

// Get extracts the pipelinerunmetrics.Recorder from the context.
func Get(ctx context.Context) *Recorder {
	untyped := ctx.Value(RecorderKey{})
	if untyped == nil {
		logging.FromContext(ctx).Panic("Unable to fetch *metrics.Recorder from context.")
	}
	return untyped.(*Recorder)
}

type recorderInformer struct {
	ctx     context.Context
	metrics *Recorder
	listers
}

// InformerKey is used for associating the Informer inside the context.Context.
type InformerKey struct{}

func WithInformer(ctx context.Context) (context.Context, controller.Informer) {
	return ctx, &recorderInformer{
		ctx:     ctx,
		metrics: Get(ctx),
		listers: listers{
			el:  elInformer.Get(ctx).Lister(),
			ctb: ctbInformer.Get(ctx).Lister(),
			tb:  tbInformer.Get(ctx).Lister(),
			tt:  ttInformer.Get(ctx).Lister(),
			ci:  ciInformer.Get(ctx).Lister(),
		},
	}
}

var _ controller.Informer = (*recorderInformer)(nil)

func (ri *recorderInformer) Run(stopCh <-chan struct{}) {
	// Turn the stopCh into a context for reporting metrics.
	ctx, cancel := context.WithCancel(ri.ctx)
	go func() {
		<-stopCh
		cancel()
	}()

	go ri.metrics.ReportCountMetrics(ctx, ri.listers)
}

func (ri *recorderInformer) HasSynced() bool {
	return true
}
