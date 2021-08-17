package sink

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/pkg/metrics"
)

var (
	elDuration = stats.Float64(
		"http_duration_seconds",
		"The eventlistener HTTP request duration",
		stats.UnitDimensionless)
	elDistribution = view.Distribution(metrics.BucketsNBy10(0.001, 5)...)
	eventCount     = stats.Float64("event_count",
		"number of events received by sink",
		stats.UnitDimensionless)
	triggeredResources = stats.Int64("triggered_resources", "Count of the number of triggered eventlistener resources", stats.UnitDimensionless)
)

const (
	failTag    = "failed"
	successTag = "succeeded"
)

// NewRecorder creates a new metrics recorder instance
// to log the TaskRun related metrics
func NewRecorder() (*Recorder, error) {
	r := &Recorder{
		initialized: true,

		// Default to reporting metrics every 30s.
		ReportingPeriod: 30 * time.Second,
	}

	status, err := tag.NewKey("status")
	if err != nil {
		return nil, err
	}
	r.status = status
	kind, err := tag.NewKey("kind")
	if err != nil {
		return nil, err
	}
	r.kind = kind

	err = view.Register(
		&view.View{
			Description: elDuration.Description(),
			Measure:     elDuration,
			Aggregation: elDistribution,
		},
		&view.View{
			Description: triggeredResources.Description(),
			Measure:     triggeredResources,
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{r.kind},
		},
		&view.View{
			Description: eventCount.Description(),
			Measure:     eventCount,
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{r.status},
		},
	)
	if err != nil {
		log.Fatalf("unable to register eventlistener metrics: %s", err)
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
	ctx, err := tag.New(
		context.Background(),
	)

	if err != nil {
		s.Logger.Warnf("failed to create tag for http metric request: %w", err)
		return
	}

	metrics.Record(ctx, elDuration.M(duration))
}

func (s *Sink) recordCountMetrics(status string) {

	s.Logger.Debugw("event listener request", "status", status)
	ctx, err := tag.New(
		context.Background(),
		tag.Insert(s.Recorder.status, status),
	)

	if err != nil {
		s.Logger.Warnf("failed to create tag for metric event_count: %w", err)
		return
	}

	metrics.Record(ctx, eventCount.M(1))
}

func (s *Sink) recordResourceCreation(resources []json.RawMessage) {
	for _, rt := range resources {
		// Assume the TriggerResourceTemplate is valid (it has an apiVersion and Kind)
		data := new(unstructured.Unstructured)
		if err := data.UnmarshalJSON(rt); err != nil {
			s.Logger.Warnf("couldn't unmarshal json from the TriggerTemplate: %v", err)
			continue
		}
		ctx, err := tag.New(context.Background(), tag.Insert(s.Recorder.kind, data.GetKind()))
		if err != nil {
			s.Logger.Warnf("failed to create tag for resource creation: %w", err)
			continue
		}

		metrics.Record(ctx, triggeredResources.M(1))
	}
}

type Recorder struct {
	initialized bool

	status tag.Key
	kind   tag.Key

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
