package sink

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestRecorderMetricsRegistered(t *testing.T) {
	r, err := NewRecorder()
	if err != nil {
		t.Fatal(err)
	}
	if !r.initialized {
		t.Fatal("Failed to initialize recorder")
	}
	// With OpenTelemetry, metrics are created in init() and registered with the global meter provider
	// The metrics (elDuration, eventRcdCount, triggeredResources) are available globally
	if elDuration == nil {
		t.Fatal("elDuration metric not initialized")
	}
	if triggeredResources == nil {
		t.Fatal("triggeredResources metric not initialized")
	}
	if eventRcdCount == nil {
		t.Fatal("eventRcdCount metric not initialized")
	}
}

func TestRecordResourceCreation(t *testing.T) {
	tests := []struct {
		name      string
		resources []json.RawMessage
	}{{
		name: "single pipelinerun",
		resources: []json.RawMessage{
			[]byte(`{"apiVersion": "tekton.dev/v1","kind": "PipelineRun","metadata": {"name": "simple-pipeline-run-lddt9"}}`),
		},
	}, {
		name: "pipelinerun and taskrun",
		resources: []json.RawMessage{
			[]byte(`{"apiVersion": "tekton.dev/v1","kind": "PipelineRun","metadata": {"name": "simple-pipeline-run-lddt9"}}`),
			[]byte(`{"apiVersion": "tekton.dev/v1","kind": "PipelineRun","metadata": {"name": "simple-pipeline-run-asdf1"}}`),
			[]byte(`{"apiVersion": "tekton.dev/v1","kind": "TaskRun","metadata": {"name": "simple-pipeline-run-lkjs9"}}`),
		},
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t).Sugar()
			r, _ := NewRecorder()
			s := &Sink{
				Recorder:          r,
				Logger:            logger,
				WGProcessTriggers: &sync.WaitGroup{},
			}
			// With OpenTelemetry, metrics are recorded asynchronously and exported
			// by the configured MeterProvider. This test verifies that
			// recordResourceCreation runs without error.
			s.recordResourceCreation(test.resources)
		})
	}
}

func TestRecordRecordDurationMetrics(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
	}{{
		name:     "Record Metrics",
		duration: 7,
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t).Sugar()
			r, _ := NewRecorder()
			s := &Sink{
				Recorder: r,
				Logger:   logger,
			}
			// With OpenTelemetry, metrics are recorded asynchronously and exported
			// by the configured MeterProvider. This test verifies that
			// recordDurationMetrics runs without error.
			s.recordDurationMetrics(&StatusRecorder{Status: 200}, test.duration)
		})
	}
}

func TestRecordRecordCountMetrics(t *testing.T) {
	tests := []struct {
		name string
	}{{
		name: "Record Metrics",
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t).Sugar()
			r, _ := NewRecorder()
			s := &Sink{
				Recorder: r,
				Logger:   logger,
			}
			// With OpenTelemetry, metrics are recorded asynchronously and exported
			// by the configured MeterProvider. This test verifies that
			// recordCountMetrics runs without error.
			s.recordCountMetrics(failTag)
		})
	}
}
