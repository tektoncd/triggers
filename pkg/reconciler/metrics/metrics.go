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
	"fmt"
	"sync"
	"time"

	"github.com/tektoncd/triggers/pkg/client/listers/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/client/listers/triggers/v1beta1"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"k8s.io/apimachinery/pkg/labels"
	"knative.dev/pkg/logging"
)

var (
	elMetricsName = "controller_eventlistener_count"
	elCount       metric.Float64Gauge

	tbMetricsName = "controller_triggerbinding_count"
	tbCount       metric.Float64Gauge

	ctbMetricsName = "controller_clustertriggerbinding_count"
	ctbCount       metric.Float64Gauge

	ttMetricsName = "controller_triggertemplate_count"
	ttCount       metric.Float64Gauge

	ciMetricsName = "controller_clusterinterceptor_count"
	ciCount       metric.Float64Gauge
)

func initMetrics() error {
	meter := otel.Meter("github.com/tektoncd/triggers/pkg/reconciler/metrics")

	var err error
	elCount, err = meter.Float64Gauge(
		elMetricsName,
		metric.WithDescription("number of eventlistener"),
	)
	if err != nil {
		return fmt.Errorf("failed to create elCount gauge: %w", err)
	}

	tbCount, err = meter.Float64Gauge(
		tbMetricsName,
		metric.WithDescription("number of triggerbinding"),
	)
	if err != nil {
		return fmt.Errorf("failed to create tbCount gauge: %w", err)
	}

	ctbCount, err = meter.Float64Gauge(
		ctbMetricsName,
		metric.WithDescription("number of clustertriggerbinding"),
	)
	if err != nil {
		return fmt.Errorf("failed to create ctbCount gauge: %w", err)
	}

	ttCount, err = meter.Float64Gauge(
		ttMetricsName,
		metric.WithDescription("number of triggertemplate"),
	)
	if err != nil {
		return fmt.Errorf("failed to create ttCount gauge: %w", err)
	}

	ciCount, err = meter.Float64Gauge(
		ciMetricsName,
		metric.WithDescription("number of clusterinterceptor"),
	)
	if err != nil {
		return fmt.Errorf("failed to create ciCount gauge: %w", err)
	}

	return nil
}

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
	once           sync.Once
	r              *Recorder
	errInitMetrics error
)

// revive:disable:unused-parameter

// NewRecorder creates a new metrics recorder instance
func NewRecorder(ctx context.Context) (*Recorder, error) {
	once.Do(func() {
		errInitMetrics = initMetrics()
		if errInitMetrics != nil {
			return
		}
		r = &Recorder{
			initialized:     true,
			ReportingPeriod: 60 * time.Second,
		}
	})

	return r, errInitMetrics
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
		r.recordGaugeMetrics(ctx, float64(count), elCount)
	}
	ci, err := li.ci.List(labels.Everything())
	if err != nil {
		logger.Errorf("error reporting trigger metrics for clusterinterceptor: %v", err)
	} else {
		count := len(ci)
		r.recordGaugeMetrics(ctx, float64(count), ciCount)
	}
	tb, err := li.tb.List(labels.Everything())
	if err != nil {
		logger.Errorf("error reporting trigger metrics for triggerbindings : %v", err)
	} else {
		count := len(tb)
		r.recordGaugeMetrics(ctx, float64(count), tbCount)
	}
	ctb, err := li.ctb.List(labels.Everything())
	if err != nil {
		logger.Errorf("error reporting trigger metrics for clustertriggerbindings: %v", err)
	} else {
		count := len(ctb)
		r.recordGaugeMetrics(ctx, float64(count), ctbCount)
	}
	tt, err := li.tt.List(labels.Everything())
	if err != nil {
		logger.Errorf("error reporting trigger metrics for triggertemplates: %v", err)
	} else {
		count := len(tt)
		r.recordGaugeMetrics(ctx, float64(count), ttCount)
	}
}

func (r *Recorder) recordGaugeMetrics(ctx context.Context, count float64, gauge metric.Float64Gauge) {
	logger := logging.FromContext(ctx)

	if !r.initialized {
		logger.Errorf("ignoring the metrics recording, failed to initialize the metrics recorder")
		return
	}

	gauge.Record(ctx, count)
}
