/*
Copyright 2025 The Knative Authors

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
	"errors"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

type shutdownFunc func(ctx context.Context) error

func noopFunc(context.Context) error { return nil }

type MeterProvider struct {
	metric.MeterProvider
	shutdown []shutdownFunc
}

func (m *MeterProvider) Shutdown(ctx context.Context) error {
	var errs []error
	for _, shutdown := range m.shutdown {
		if err := shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func NewMeterProvider(
	ctx context.Context,
	cfg Config,
) (*MeterProvider, error) {
	if cfg.Protocol == ProtocolNone {
		return &MeterProvider{MeterProvider: noop.NewMeterProvider()}, nil
	}

	// For now, return a noop provider
	// In a full implementation, you would configure the appropriate provider
	// based on the protocol (gRPC, HTTP, Prometheus)
	return &MeterProvider{
		MeterProvider: noop.NewMeterProvider(),
		shutdown:      []shutdownFunc{noopFunc},
	}, nil
}
