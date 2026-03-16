package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	elDuration         metric.Float64Histogram
	eventRcdCount      metric.Int64Counter
	triggeredResources metric.Int64Counter
)

const (
	failTag    = "failed"
	successTag = "succeeded"
)

var (
	once           sync.Once
	errInitMetrics error
)

func initMetrics() error {
	meter := otel.Meter("github.com/tektoncd/triggers/pkg/sink")

	var err error
	elDuration, err = meter.Float64Histogram(
		"eventlistener_http_duration_seconds",
		metric.WithDescription("The eventlistener HTTP request duration"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.01, 0.1, 1, 5, 10),
	)
	if err != nil {
		return fmt.Errorf("failed to create elDuration histogram: %w", err)
	}

	eventRcdCount, err = meter.Int64Counter(
		"eventlistener_event_received_total",
		metric.WithDescription("number of events received by sink"),
	)
	if err != nil {
		return fmt.Errorf("failed to create eventRcdCount counter: %w", err)
	}

	triggeredResources, err = meter.Int64Counter(
		"eventlistener_triggered_resources_total",
		metric.WithDescription("Count of the number of triggered eventlistener resources"),
	)
	if err != nil {
		return fmt.Errorf("failed to create triggeredResources counter: %w", err)
	}

	return nil
}

// NewRecorder creates a new metrics recorder instance
func NewRecorder() (*Recorder, error) {
	once.Do(func() {
		errInitMetrics = initMetrics()
	})
	if errInitMetrics != nil {
		return nil, errInitMetrics
	}

	r := &Recorder{
		initialized:     true,
		ReportingPeriod: 30 * time.Second,
	}

	return r, nil
}

func (s *Sink) NewMetricsRecorderInterceptor() MetricsInterceptor {
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		recorder := &StatusRecorder{
			ResponseWriter: w,
			Status:         200,
		}
		startTime := time.Now()
		defer func() {
			endTime := time.Now()
			elapsed := endTime.Sub(startTime)
			// Log the consumed time
			go s.recordDurationMetrics(recorder, elapsed)
		}()
		next(recorder, r)
	}
}

func (s *Sink) recordDurationMetrics(w *StatusRecorder, elapsed time.Duration) {
	duration := elapsed.Seconds()
	s.Logger.Debugw("event listener request completed", "status", w.Status, "duration", duration)

	elDuration.Record(context.Background(), duration)
}

func (s *Sink) recordCountMetrics(status string) {
	s.Logger.Debugw("event listener request", "status", status)

	eventRcdCount.Add(context.Background(), 1, metric.WithAttributes(attribute.String("status", status)))
}

func (s *Sink) recordResourceCreation(resources []json.RawMessage) {
	for _, rt := range resources {
		// Assume the TriggerResourceTemplate is valid (it has an apiVersion and Kind)
		data := new(unstructured.Unstructured)
		if err := data.UnmarshalJSON(rt); err != nil {
			s.Logger.Warnf("couldn't unmarshal json from the TriggerTemplate: %v", err)
			continue
		}

		triggeredResources.Add(context.Background(), 1, metric.WithAttributes(attribute.String("kind", data.GetKind())))
	}
}

type Recorder struct {
	initialized bool

	ReportingPeriod time.Duration
}

type StatusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *StatusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

// MetricsInterceptor intercepts an HTTP handler invocation, it is passed both response writer and request
// which after interception can be passed onto the handler function.
type MetricsInterceptor func(http.ResponseWriter, *http.Request, http.HandlerFunc)

// MetricsHandlerFunc builds on top of http.Handler, and exposes API to intercept with MetricsInterceptor.
// This allows building complex long chains without complicated struct manipulation
type MetricsHandler struct {
	Handler http.Handler
}

// Intercept returns back a continuation that will call the handler func to intercept
// the continuation call.
func (cont *MetricsHandler) Intercept(mw MetricsInterceptor) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		mw(writer, request, cont.Handler.ServeHTTP)
	}
}
