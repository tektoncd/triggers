package sdk

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"
)

// TODO: Use sharedmain.MainWithConfig to simplify this
func InterceptorMainWithConfig(ctx context.Context, component string, interceptors map[string]InterceptorFunc) {
	cfg := injection.ParseAndGetRESTConfigOrDie()
	ctx, _ = injection.EnableInjectionOrDie(ctx, cfg)

	logger, atomicLevel := sharedmain.SetupLoggerOrDie(ctx, component)

	ctx = logging.WithLogger(ctx, logger)
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Fatalf("failed to sync the logger: %s", err)
		}
	}()
	ctx = logging.WithLogger(ctx, logger)

	cmw := sharedmain.SetupConfigMapWatchOrDie(ctx, logger)

	sharedmain.WatchLoggingConfigOrDie(ctx, cmw, logger, atomicLevel, component)
	// Register logger
	// Register injection context

	service, err := NewWithInterceptors(ctx, interceptors)
	if err != nil {
		log.Fatalf("failed to initialize core interceptors: %s", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", service)
	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "ok")
	})

	options := GetOptions(ctx)
	if options == nil {
		options = &Options{}
	}
	setDefaultOptions(options)
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", options.Port),
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      mux,
	}

	logger.Infof("Listen and serve on port %d", options.Port)
	if err := srv.ListenAndServe(); err != nil {
		logger.Fatalf("failed to start interceptors service: %v", err)
	}
}

// Options contains the configuration for the interceptor
type Options struct {
	Port int
}

// optionsKey is used as the key for associating information
// with a context.Context.
type optionsKey struct{}

func setDefaultOptions(opt *Options) {
	if opt.Port == 0 {
		// Default port is 8082
		opt.Port = 8082
	}
}

// WithOptions associates a set of webhook.Options with
// the returned context.
func WithOptions(ctx context.Context, opt Options) context.Context {
	return context.WithValue(ctx, optionsKey{}, &opt)
}

// GetOptions retrieves webhook.Options associated with the
// given context via WithOptions (above).
func GetOptions(ctx context.Context) *Options {
	v := ctx.Value(optionsKey{})
	if v == nil {
		return nil
	}
	return v.(*Options)
}
