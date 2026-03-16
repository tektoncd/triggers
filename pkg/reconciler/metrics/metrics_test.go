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
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	fakeCIInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clusterinterceptor/fake"
	fakeCTBInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/clustertriggerbinding/fake"
	fakeELInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/eventlistener/fake"
	fakeTBInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/triggerbinding/fake"
	fakeTTInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/triggertemplate/fake"
	"github.com/tektoncd/triggers/test"
)

func unregisterMetrics() {
	once = sync.Once{}
	r = nil
	errInitMetrics = nil
	elCount = nil
	tbCount = nil
	ctbCount = nil
	ttCount = nil
	ciCount = nil
}

func setupTestProvider(t *testing.T) *sdkmetric.ManualReader {
	t.Helper()
	unregisterMetrics()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)
	t.Cleanup(func() { provider.Shutdown(context.Background()) })
	return reader
}

func collectMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Collect error: %v", err)
	}
	return rm
}

func findMetric(rm metricdata.ResourceMetrics, name string) (metricdata.Metrics, bool) {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m, true
			}
		}
	}
	return metricdata.Metrics{}, false
}

func TestUninitializedMetrics(t *testing.T) {
	metrics := &Recorder{}
	ctx, _ := test.SetupFakeContext(t)

	metrics.recordGaugeMetrics(ctx, 3, elCount)
	metrics.recordGaugeMetrics(ctx, 3, ctbCount)
	metrics.recordGaugeMetrics(ctx, 3, tbCount)
	metrics.recordGaugeMetrics(ctx, 3, ttCount)
	metrics.recordGaugeMetrics(ctx, 3, ciCount)
}

func TestCountMetrics(t *testing.T) {
	reader := setupTestProvider(t)
	ctx, _ := test.SetupFakeContext(t)
	ctx = WithClient(ctx)

	rec := Get(ctx)

	fakeELIn := fakeELInformer.Get(ctx)
	fakeCTBIn := fakeCTBInformer.Get(ctx)
	fakeTBIn := fakeTBInformer.Get(ctx)
	fakeTTIn := fakeTTInformer.Get(ctx)
	fakeCIIn := fakeCIInformer.Get(ctx)

	for _, el := range []*v1beta1.EventListener{
		{ObjectMeta: metav1.ObjectMeta{Name: "el-1", Namespace: "test"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "el-2", Namespace: "test"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "el-3", Namespace: "test"}},
	} {
		if err := fakeELIn.Informer().GetIndexer().Add(el); err != nil {
			t.Fatalf("Adding EL to informer: %v", err)
		}
	}

	for _, tt := range []*v1beta1.TriggerTemplate{
		{ObjectMeta: metav1.ObjectMeta{Name: "tt-1", Namespace: "test"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "tt-2", Namespace: "test"}},
	} {
		if err := fakeTTIn.Informer().GetIndexer().Add(tt); err != nil {
			t.Fatalf("Adding TT to informer: %v", err)
		}
	}

	for _, tb := range []*v1beta1.TriggerBinding{
		{ObjectMeta: metav1.ObjectMeta{Name: "tb-1", Namespace: "test"}},
	} {
		if err := fakeTBIn.Informer().GetIndexer().Add(tb); err != nil {
			t.Fatalf("Adding TB to informer: %v", err)
		}
	}

	for _, ctb := range []*v1beta1.ClusterTriggerBinding{
		{ObjectMeta: metav1.ObjectMeta{Name: "ctb-1", Namespace: "test"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "ctb-2", Namespace: "test"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "ctb-3", Namespace: "test"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "ctb-4", Namespace: "test"}},
	} {
		if err := fakeCTBIn.Informer().GetIndexer().Add(ctb); err != nil {
			t.Fatalf("Adding CTB to informer: %v", err)
		}
	}

	for _, ci := range []*v1alpha1.ClusterInterceptor{
		{ObjectMeta: metav1.ObjectMeta{Name: "ci-1", Namespace: "test"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "ci-2", Namespace: "test"}},
	} {
		if err := fakeCIIn.Informer().GetIndexer().Add(ci); err != nil {
			t.Fatalf("Adding CI to informer: %v", err)
		}
	}

	li := listers{
		el:  fakeELIn.Lister(),
		ctb: fakeCTBIn.Lister(),
		tb:  fakeTBIn.Lister(),
		tt:  fakeTTIn.Lister(),
		ci:  fakeCIIn.Lister(),
	}

	rec.CountMetrics(ctx, li)

	rm := collectMetrics(t, reader)

	for _, tc := range []struct {
		name  string
		value float64
	}{
		{elMetricsName, 3},
		{ttMetricsName, 2},
		{tbMetricsName, 1},
		{ctbMetricsName, 4},
		{ciMetricsName, 2},
	} {
		m, found := findMetric(rm, tc.name)
		if !found {
			t.Errorf("metric %s not found", tc.name)
			continue
		}
		gauge, ok := m.Data.(metricdata.Gauge[float64])
		if !ok {
			t.Errorf("metric %s: expected Gauge[float64], got %T", tc.name, m.Data)
			continue
		}
		if len(gauge.DataPoints) != 1 {
			t.Errorf("metric %s: expected 1 data point, got %d", tc.name, len(gauge.DataPoints))
			continue
		}
		if gauge.DataPoints[0].Value != tc.value {
			t.Errorf("metric %s: expected %v, got %v", tc.name, tc.value, gauge.DataPoints[0].Value)
		}
	}
}

func TestIndividualGaugeCounts(t *testing.T) {
	for _, tc := range []struct {
		name       string
		metricName string
		gauge      *metric.Float64Gauge
	}{
		{"eventlistener", elMetricsName, &elCount},
		{"triggertemplate", ttMetricsName, &ttCount},
		{"triggerbinding", tbMetricsName, &tbCount},
		{"clustertriggerbinding", ctbMetricsName, &ctbCount},
		{"clusterinterceptor", ciMetricsName, &ciCount},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := setupTestProvider(t)
			ctx, _ := test.SetupFakeContext(t)
			ctx = WithClient(ctx)

			rec := Get(ctx)
			rec.recordGaugeMetrics(ctx, 5, *tc.gauge)

			rm := collectMetrics(t, reader)
			m, found := findMetric(rm, tc.metricName)
			if !found {
				t.Fatalf("metric %s not found", tc.metricName)
			}
			gauge, ok := m.Data.(metricdata.Gauge[float64])
			if !ok {
				t.Fatalf("expected Gauge[float64], got %T", m.Data)
			}
			if len(gauge.DataPoints) != 1 {
				t.Fatalf("expected 1 data point, got %d", len(gauge.DataPoints))
			}
			if gauge.DataPoints[0].Value != 5 {
				t.Errorf("expected 5, got %v", gauge.DataPoints[0].Value)
			}
		})
	}
}
