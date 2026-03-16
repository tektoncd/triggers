package sink

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap/zaptest"
)

func resetSinkMetrics() {
	once = sync.Once{}
	errInitMetrics = nil
	elDuration = nil
	eventRcdCount = nil
	triggeredResources = nil
}

func setupTestProvider(t *testing.T) *sdkmetric.ManualReader {
	t.Helper()
	resetSinkMetrics()
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

func TestRecorderMetricsRegistered(t *testing.T) {
	reader := setupTestProvider(t)

	r, err := NewRecorder()
	if err != nil {
		t.Fatal(err)
	}
	if !r.initialized {
		t.Fatal("Failed to initialize recorder")
	}
	if elDuration == nil {
		t.Fatal("elDuration metric not initialized")
	}
	if triggeredResources == nil {
		t.Fatal("triggeredResources metric not initialized")
	}
	if eventRcdCount == nil {
		t.Fatal("eventRcdCount metric not initialized")
	}

	_ = reader
}

func TestRecordCountMetrics(t *testing.T) {
	for _, tc := range []struct {
		name           string
		status         string
		expectedStatus string
		expectedValue  int64
	}{{
		name:           "succeeded event",
		status:         successTag,
		expectedStatus: "succeeded",
		expectedValue:  1,
	}, {
		name:           "failed event",
		status:         failTag,
		expectedStatus: "failed",
		expectedValue:  1,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			reader := setupTestProvider(t)

			_, err := NewRecorder()
			if err != nil {
				t.Fatal(err)
			}
			logger := zaptest.NewLogger(t).Sugar()
			s := &Sink{
				Recorder: &Recorder{initialized: true},
				Logger:   logger,
			}

			s.recordCountMetrics(tc.status)

			rm := collectMetrics(t, reader)
			m, found := findMetric(rm, "eventlistener_event_received_total")
			if !found {
				t.Fatal("eventlistener_event_received_total metric not found")
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("expected Sum[int64], got %T", m.Data)
			}
			if len(sum.DataPoints) != 1 {
				t.Fatalf("expected 1 data point, got %d", len(sum.DataPoints))
			}
			dp := sum.DataPoints[0]
			if dp.Value != tc.expectedValue {
				t.Errorf("expected count %d, got %d", tc.expectedValue, dp.Value)
			}

			gotAttrs := make(map[string]string)
			for _, kv := range dp.Attributes.ToSlice() {
				gotAttrs[string(kv.Key)] = kv.Value.AsString()
			}
			wantAttrs := map[string]string{"status": tc.expectedStatus}
			if d := cmp.Diff(wantAttrs, gotAttrs); d != "" {
				t.Errorf("attributes diff (-want, +got): %s", d)
			}
		})
	}
}

func TestRecordDurationMetrics(t *testing.T) {
	reader := setupTestProvider(t)

	_, err := NewRecorder()
	if err != nil {
		t.Fatal(err)
	}
	logger := zaptest.NewLogger(t).Sugar()
	s := &Sink{
		Recorder: &Recorder{initialized: true},
		Logger:   logger,
	}

	s.recordDurationMetrics(&StatusRecorder{Status: 200}, 7*time.Second)

	rm := collectMetrics(t, reader)
	m, found := findMetric(rm, "eventlistener_http_duration_seconds")
	if !found {
		t.Fatal("eventlistener_http_duration_seconds metric not found")
	}
	hist, ok := m.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("expected Histogram[float64], got %T", m.Data)
	}
	if len(hist.DataPoints) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(hist.DataPoints))
	}
	dp := hist.DataPoints[0]
	if dp.Sum != 7.0 {
		t.Errorf("expected duration sum 7.0, got %v", dp.Sum)
	}
	if dp.Count != 1 {
		t.Errorf("expected count 1, got %d", dp.Count)
	}
}

func TestRecordResourceCreation(t *testing.T) {
	for _, tc := range []struct {
		name          string
		resources     []json.RawMessage
		expectedKinds map[string]int64
	}{{
		name: "single pipelinerun",
		resources: []json.RawMessage{
			[]byte(`{"apiVersion": "tekton.dev/v1","kind": "PipelineRun","metadata": {"name": "pr-1"}}`),
		},
		expectedKinds: map[string]int64{"PipelineRun": 1},
	}, {
		name: "mixed resources",
		resources: []json.RawMessage{
			[]byte(`{"apiVersion": "tekton.dev/v1","kind": "PipelineRun","metadata": {"name": "pr-1"}}`),
			[]byte(`{"apiVersion": "tekton.dev/v1","kind": "PipelineRun","metadata": {"name": "pr-2"}}`),
			[]byte(`{"apiVersion": "tekton.dev/v1","kind": "TaskRun","metadata": {"name": "tr-1"}}`),
		},
		expectedKinds: map[string]int64{"PipelineRun": 2, "TaskRun": 1},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			reader := setupTestProvider(t)

			_, err := NewRecorder()
			if err != nil {
				t.Fatal(err)
			}
			logger := zaptest.NewLogger(t).Sugar()
			s := &Sink{
				Recorder:          &Recorder{initialized: true},
				Logger:            logger,
				WGProcessTriggers: &sync.WaitGroup{},
			}

			s.recordResourceCreation(tc.resources)

			rm := collectMetrics(t, reader)
			m, found := findMetric(rm, "eventlistener_triggered_resources_total")
			if !found {
				t.Fatal("eventlistener_triggered_resources_total metric not found")
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("expected Sum[int64], got %T", m.Data)
			}

			gotKinds := make(map[string]int64)
			for _, dp := range sum.DataPoints {
				for _, kv := range dp.Attributes.ToSlice() {
					if string(kv.Key) == "kind" {
						gotKinds[kv.Value.AsString()] = dp.Value
					}
				}
			}
			if d := cmp.Diff(tc.expectedKinds, gotKinds); d != "" {
				t.Errorf("eventlistener_triggered_resources_total diff (-want, +got): %s", d)
			}
		})
	}
}
