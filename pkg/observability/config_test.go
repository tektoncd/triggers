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

package observability

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/observability/metrics"
	"github.com/tektoncd/triggers/pkg/observability/runtime"
	"github.com/tektoncd/triggers/pkg/observability/tracing"
)

func TestConfigInContext(t *testing.T) {
	want := DefaultConfig()
	ctx := WithConfig(context.Background(), want)
	got := GetConfig(ctx)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestNoConfigInContext(t *testing.T) {
	got := GetConfig(context.Background())
	want := (*Config)(nil)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestNewFromMap(t *testing.T) {
	cm := map[string]string{
		"tracing-protocol":        "grpc",
		"tracing-endpoint":        "https://example.com",
		"tracing-sampling-rate":   "1",
		"runtime-profiling":       "enabled",
		"runtime-export-interval": "15s",
		"metrics-protocol":        "grpc",
		"metrics-endpoint":        "https://example.com",
		"metrics-export-interval": "10s",
	}

	got, err := NewFromMap(cm)
	if err != nil {
		t.Error("unexpected error: ", err)
	}

	want := &Config{
		Tracing: tracing.Config{
			Protocol:     "grpc",
			Endpoint:     "https://example.com",
			SamplingRate: 1,
		},
		Runtime: runtime.Config{
			Profiling:      "enabled",
			ExportInterval: 15 * time.Second,
		},
		Metrics: metrics.Config{
			Protocol:       "grpc",
			Endpoint:       "https://example.com",
			ExportInterval: 10 * time.Second,
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
