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
	"sync"
	"time"

	"github.com/tektoncd/triggers/pkg/client/listers/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/client/listers/triggers/v1beta1"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"k8s.io/apimachinery/pkg/labels"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/metrics"
)

var (
	elMetricsName = "eventlistener_count"
	elCount       = stats.Float64(elMetricsName,
		"number of eventlistener",
		stats.UnitDimensionless)
	elCountView *view.View

	tbMetricsName = "triggerbinding_count"
	tbCount       = stats.Float64(tbMetricsName,
		"number of triggerbinding",
		stats.UnitDimensionless)
	tbCountView *view.View

	ctbMetricsName = "clustertriggerbinding_count"
	ctbCount       = stats.Float64(ctbMetricsName,
		"number of clustertriggerbinding",
		stats.UnitDimensionless)
	ctbCountView *view.View

	ttMetricsName = "triggertemplate_count"
	ttCount       = stats.Float64(ttMetricsName,
		"number of triggertemplate",
		stats.UnitDimensionless)
	ttCountView *view.View

	ciMetricsName = "clusterinterceptor_count"
	ciCount       = stats.Float64(ciMetricsName,
		"number of clusterinterceptor",
		stats.UnitDimensionless)
	ciCountView *view.View
)

type listers struct {
	el  v1beta1.EventListenerLister
	ctb v1beta1.ClusterTriggerBindingLister
	tb  v1beta1.TriggerBindingLister
	tt  v1beta1.TriggerTemplateLister
	ci  v1alpha1.ClusterInterceptorLister
}

// Recorder holds information for Trigger metrics
type Recorder struct {
	initialized     bool
	ReportingPeriod time.Duration
}

// We cannot register the view multiple times, so NewRecorder lazily
// initializes this singleton and returns the same recorder across any
// subsequent invocations.
var (
	once        sync.Once
	r           *Recorder
	recorderErr error
)

// revive:disable:unused-parameter

// NewRecorder creates a new metrics recorder instance
// to log the PipelineRun related metrics
func NewRecorder(ctx context.Context) (*Recorder, error) {
	once.Do(func() {
		r = &Recorder{
			initialized: true,
			// Default to reporting metrics every 60s.
			ReportingPeriod: 60 * time.Second,
		}

		recorderErr = viewRegister()
		if recorderErr != nil {
			r.initialized = false
			return
		}
	})

	return r, recorderErr
}

func viewRegister() error {
	elCountView = &view.View{
		Description: elCount.Description(),
		Measure:     elCount,
		Aggregation: view.LastValue(),
	}

	tbCountView = &view.View{
		Description: tbCount.Description(),
		Measure:     tbCount,
		Aggregation: view.LastValue(),
	}

	ctbCountView = &view.View{
		Description: ctbCount.Description(),
		Measure:     ctbCount,
		Aggregation: view.LastValue(),
	}

	ttCountView = &view.View{
		Description: ttCount.Description(),
		Measure:     ttCount,
		Aggregation: view.LastValue(),
	}

	ciCountView = &view.View{
		Description: ciCount.Description(),
		Measure:     ciCount,
		Aggregation: view.LastValue(),
	}

	return view.Register(
		elCountView,
		tbCountView,
		ctbCountView,
		ttCountView,
		ciCountView,
	)
}

func (r *Recorder) ReportCountMetrics(ctx context.Context, li listers) {

	for {
		select {
		case <-ctx.Done():
			// When the context is cancelled, stop reporting.
			return

		case <-time.After(r.ReportingPeriod):
			r.CountMetrics(ctx, li)
		}
	}
}

func (r *Recorder) CountMetrics(ctx context.Context, li listers) {
	logger := logging.FromContext(ctx)

	el, err := li.el.List(labels.Everything())
	if err != nil {
		logger.Errorf("error reporting trigger metrics for eventlisteners: %v", err)
	} else {
		count := len(el)
		r.countMetrics(ctx, float64(count), elCount)
	}
	ci, err := li.ci.List(labels.Everything())
	if err != nil {
		logger.Errorf("error reporting trigger metrics for clusterinterceptor: %v", err)
	} else {
		count := len(ci)
		r.countMetrics(ctx, float64(count), ciCount)
	}
	tb, err := li.tb.List(labels.Everything())
	if err != nil {
		logger.Errorf("error reporting trigger metrics for triggerbindings : %v", err)
	} else {
		count := len(tb)
		r.countMetrics(ctx, float64(count), tbCount)
	}
	ctb, err := li.ctb.List(labels.Everything())
	if err != nil {
		logger.Errorf("error reporting trigger metrics for clustertriggerbindings: %v", err)
	} else {
		count := len(ctb)
		r.countMetrics(ctx, float64(count), ctbCount)
	}
	tt, err := li.tt.List(labels.Everything())
	if err != nil {
		logger.Errorf("error reporting trigger metrics for triggertemplates: %v", err)
	} else {
		count := len(tt)
		r.countMetrics(ctx, float64(count), ttCount)
	}

}

func (r *Recorder) countMetrics(ctx context.Context, count float64, measure *stats.Float64Measure) {
	logger := logging.FromContext(ctx)

	if !r.initialized {
		logger.Errorf("ignoring the metrics recording for %s, failed to initialize the metrics recorder", measure.Description())
	}

	metrics.Record(ctx, measure.M(count))
}
