package sink

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"go.opencensus.io/stats/view"
	"go.uber.org/zap/zaptest"
	"knative.dev/pkg/metrics"
	"knative.dev/pkg/metrics/metricstest"
)

func TestRecorderMetricsRegistered(t *testing.T) {
	r, err := NewRecorder()
	if err != nil {
		t.Fatal(err)
	}
	if !r.initialized {
		t.Fatal("Failed to initialize recorder")
	}
	v := view.Find("http_duration_seconds")
	if v == nil {
		t.Fatal("Unable to find http_duration_seconds metric")
	}
	v = view.Find("triggered_resources")
	if v == nil {
		t.Fatal("Unable to find triggered_resources metric")
	}
}

func TestRecordResourceCreation(t *testing.T) {
	tests := []struct {
		name           string
		resources      []json.RawMessage
		resourceCounts map[string]int64
	}{{
		name: "single pipelinerun",
		resources: []json.RawMessage{
			[]byte(`{"apiVersion": "tekton.dev/v1","kind": "PipelineRun","metadata": {"name": "simple-pipeline-run-lddt9"}}`),
		},
		resourceCounts: map[string]int64{
			"PipelineRun": 1,
		},
	}, {
		name: "pipelinerun and taskrun",
		resources: []json.RawMessage{
			[]byte(`{"apiVersion": "tekton.dev/v1","kind": "PipelineRun","metadata": {"name": "simple-pipeline-run-lddt9"}}`),
			[]byte(`{"apiVersion": "tekton.dev/v1","kind": "PipelineRun","metadata": {"name": "simple-pipeline-run-asdf1"}}`),
			[]byte(`{"apiVersion": "tekton.dev/v1","kind": "TaskRun","metadata": {"name": "simple-pipeline-run-lkjs9"}}`),
		},
		resourceCounts: map[string]int64{
			"PipelineRun": 2,
			"TaskRun":     1,
		},
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t).Sugar()
			metrics.FlushExporter()
			err := metrics.UpdateExporter(context.TODO(), metrics.ExporterOptions{
				Domain:    "tekton.dev/triggers",
				Component: "triggers",
				ConfigMap: map[string]string{},
			}, logger)
			if err != nil {
				t.Fatal(err)
			}
			r, _ := NewRecorder()
			s := &Sink{
				Recorder:          r,
				WGProcessTriggers: &sync.WaitGroup{},
			}
			s.recordResourceCreation(test.resources)
			rows, err := view.RetrieveData("triggered_resources")
			if err != nil {
				t.Fatal(err)
			}
			for k, v := range test.resourceCounts {
				found := false
				for _, row := range rows {
					if row.Tags[0].Value == k {
						found = true
						if row.Data.(*view.CountData).Value != v {
							t.Fatalf("Expected %d resources of kind %s, found %d", v, k, row.Data.(*view.CountData).Value)
						}
						break
					}
				}
				if !found {
					t.Fatalf("Expected resources recorded of kind %s, received none", k)
				}
			}
			v := view.Find("triggered_resources")
			// need to unregister the view so the counts reset
			view.Unregister(v)
		})
	}
}

func TestRecordRecordDurationMetrics(t *testing.T) {
	tests := []struct {
		name             string
		duration         time.Duration
		expectedDuration float64
	}{{
		name:             "Record Metrics",
		duration:         7,
		expectedDuration: 7e-09,
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer metricstest.Unregister("event_count", "http_duration_seconds")
			logger := zaptest.NewLogger(t).Sugar()
			metrics.FlushExporter()
			err := metrics.UpdateExporter(context.TODO(), metrics.ExporterOptions{
				Domain:    "tekton.dev/triggers",
				Component: "triggers",
				ConfigMap: map[string]string{},
			}, logger)
			if err != nil {
				t.Fatal(err)
			}
			r, _ := NewRecorder()
			s := &Sink{
				Recorder: r,
				Logger:   logger,
			}
			s.recordDurationMetrics(&StatusRecorder{Status: 200}, test.duration)
			metricstest.CheckDistributionData(t, "http_duration_seconds", nil, 1, test.expectedDuration, test.expectedDuration)
		})
	}
}

func TestRecordRecordCountMetrics(t *testing.T) {
	tests := []struct {
		name          string
		expectedTags  map[string]string
		expectedCount int64
	}{{
		name: "Record Metrics",
		expectedTags: map[string]string{
			"status": failTag,
		},
		expectedCount: 1,
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer metricstest.Unregister("event_count", "http_duration_seconds")
			logger := zaptest.NewLogger(t).Sugar()
			metrics.FlushExporter()
			err := metrics.UpdateExporter(context.TODO(), metrics.ExporterOptions{
				Domain:    "tekton.dev/triggers",
				Component: "triggers",
				ConfigMap: map[string]string{},
			}, logger)
			if err != nil {
				t.Fatal(err)
			}
			r, _ := NewRecorder()
			s := &Sink{
				Recorder: r,
				Logger:   logger,
			}
			s.recordCountMetrics(failTag)
			metricstest.CheckCountData(t, "event_count", test.expectedTags, test.expectedCount)
		})
	}
}
