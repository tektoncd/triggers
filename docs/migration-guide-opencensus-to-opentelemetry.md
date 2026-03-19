<!--
---
linkTitle: "OpenTelemetry Migration"
weight: 1201
---
-->
# Migration Guide: OpenCensus to OpenTelemetry

This guide covers changes for cluster operators upgrading Tekton Triggers
to a version that uses OpenTelemetry for metrics and tracing. The Triggers
API (EventListener, TriggerTemplate, TriggerBinding, etc.) is unchanged.

This migration aligns Triggers with
[Tekton Pipeline PR #9043](https://github.com/tektoncd/pipeline/pull/9043)
and Knative's migration from `knative.dev/pkg/metrics` (OpenCensus) to
`knative.dev/pkg/observability` (OpenTelemetry).

---

## What changed

| Area | Before | After |
|---|---|---|
| Metrics library | OpenCensus (`go.opencensus.io`) | OpenTelemetry (`go.opentelemetry.io/otel`) |
| ConfigMap key for metrics | `metrics.backend-destination` | `metrics-protocol` |
| ConfigMap key for tracing | Separate `config-tracing` / `K_TRACING_CONFIG` | Unified in `config-observability-triggers` (`tracing-protocol`) |
| Stackdriver support | Built-in | Use OTLP with a GCP exporter |
| Sink counter names | `eventlistener_event_received_count` | `eventlistener_event_received_total` |
| Sink counter names | `eventlistener_triggered_resources` | `eventlistener_triggered_resources_total` |
| All other metric names | unchanged | unchanged |
| Default metrics export | Prometheus on port 9000 | Prometheus on port 9000 (unchanged) |
| Metrics port | 9000 | 9000 (unchanged) |

---

## Step 1: Observability ConfigMap

The `config-observability-triggers` ConfigMap ships with
`metrics-protocol: prometheus` as active data, so Prometheus metrics
remain enabled by default.

The `_example` block now documents the new OpenTelemetry configuration
keys. The old OpenCensus keys (`metrics.backend-destination`,
`metrics.stackdriver-project-id`, `metrics.allow-stackdriver-custom-metrics`)
are no longer recognized.

**If you had no custom observability configuration**, no action is needed.

**If you had custom values**, translate them:

| Old key | New key |
|---|---|
| `metrics.backend-destination: prometheus` | `metrics-protocol: prometheus` |
| `metrics.backend-destination: opencensus` | `metrics-protocol: grpc` + `metrics-endpoint: <host:port>` |
| `metrics.backend-destination: none` | `metrics-protocol: none` |
| `metrics.opencensus-address` | `metrics-endpoint` |
| `metrics.reporting-period-seconds: 60` | `metrics-export-interval: 60s` |

After changing the ConfigMap, restart the controller:

```bash
kubectl -n tekton-pipelines rollout restart deployment tekton-triggers-controller
```

---

## Step 2: Update Prometheus queries

Only the two sink counter metrics were renamed. Update any queries or alerts
that reference them:

```promql
# Before
eventlistener_event_received_count{status="succeeded"}
eventlistener_triggered_resources{kind="PipelineRun"}

# After
eventlistener_event_received_total{status="succeeded"}
eventlistener_triggered_resources_total{kind="PipelineRun"}
```

All other metric names are unchanged:

```promql
# These work as before
controller_eventlistener_count
controller_clusterinterceptor_count
eventlistener_http_duration_seconds_bucket{le="1.0"}
```

---

## Step 3: Account for new OTel scope labels

All metrics now include OpenTelemetry instrumentation scope labels:

- `otel_scope_name` (e.g. `github.com/tektoncd/triggers/pkg/sink`
  for sink metrics, `github.com/tektoncd/triggers/pkg/reconciler/metrics`
  for controller gauge metrics)
- `otel_scope_schema_url` (emitted but empty)
- `otel_scope_version` (emitted but empty)

Only `otel_scope_name` carries a meaningful value. These labels are
informational and transparent to most PromQL queries. If you use strict
label matching in recording rules or alerts, account for the new
`otel_scope_name` label.

---

## Step 4: Verify

```bash
kubectl -n tekton-pipelines port-forward svc/tekton-triggers-controller 9000:9000
curl -s http://localhost:9000/metrics | grep -E "controller_eventlistener_count|eventlistener_event_received_total"
```

---

## No action required

- **Triggers API**: All CRDs are unchanged.
- **Metrics port**: Still 9000, controlled by `METRICS_PROMETHEUS_PORT`.
- **Metric semantics**: All metrics measure the same things with the same
  labels (`status`, `kind`).
- **EventListener behavior**: Events are processed identically.

---

## Rollback

Redeploy the previous Triggers release. The old OpenCensus keys will work
again. No data migration is needed.
