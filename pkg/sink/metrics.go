package sink

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"knative.dev/pkg/metrics"
)

var (
	elDuration = stats.Float64(
		"eventlistener_http_duration_seconds",
		"The eventlistener HTTP request duration",
		stats.UnitDimensionless)
	elDistribution = view.Distribution(metrics.BucketsNBy10(0.001, 5)...)
)

type Recorder struct {
	initialized bool

	status tag.Key

	ReportingPeriod time.Duration
}

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

	err = view.Register(
		&view.View{
			Description: elDuration.Description(),
			Measure:     elDuration,
			Aggregation: elDistribution,
			TagKeys:     []tag.Key{r.status},
		},
	)
	if err != nil {
		log.Fatalf("unable to register eventlistener metrics: %s", err)
	}
	return r, nil
}

type StatusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *StatusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func NewMetricsRecorderInterceptor(s Sink) MiddlewareInterceptor {
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
			go s.recordMetrics(recorder, elapsed)
		}()
		next(recorder, r)
	}
}

func (r *Sink) recordMetrics(w *StatusRecorder, elapsed time.Duration) {

	duration := elapsed.Seconds()
	r.Logger.Debugw("event listener request completed", "status", w.Status, "duration", duration)
	ctx, err := tag.New(
		context.Background(),
		tag.Insert(r.Recorder.status, strconv.Itoa(w.Status)),
	)

	if err != nil {
		r.Logger.Warnf("failed to create tag for request: %s", err)
		return
	}

	metrics.Record(ctx, elDuration.M(duration))
}

// MiddlewareInterceptor intercepts an HTTP handler invocation, it is passed both response writer and request
// which after interception can be passed onto the handler function.
type MiddlewareInterceptor func(http.ResponseWriter, *http.Request, http.HandlerFunc)

// MiddlewareHandlerFunc builds on top of http.HandlerFunc, and exposes API to intercept with MiddlewareInterceptor.
// This allows building complex long chains without complicated struct manipulation
type MiddlewareHandlerFunc http.HandlerFunc

// Intercept returns back a continuation that will call install middleware to intercept
// the continuation call.
func (cont MiddlewareHandlerFunc) Intercept(mw MiddlewareInterceptor) MiddlewareHandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		mw(writer, request, http.HandlerFunc(cont))
	}
}

// MiddlewareChain is a collection of interceptors that will be invoked in there index order
type MiddlewareChain []MiddlewareInterceptor

// Handler allows hooking multiple middleware in single call.
func (chain MiddlewareChain) Handler(handler http.HandlerFunc) http.Handler {
	curr := MiddlewareHandlerFunc(handler)
	for i := len(chain) - 1; i >= 0; i-- {
		mw := chain[i]
		curr = curr.Intercept(mw)
	}
	return http.HandlerFunc(curr)
}
