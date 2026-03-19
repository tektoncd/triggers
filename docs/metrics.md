<!--
---
linkTitle: "Triggers Metrics"
weight: 1200
---
-->
# Tekton Triggers Metrics

## Controller Metrics

The following metrics are exported by the triggers controller on port `9000`.

| Name | Type | Labels/Tags | Description |
|---|---|---|---|
| `controller_eventlistener_count` | Gauge | | Number of EventListeners |
| `controller_triggerbinding_count` | Gauge | | Number of TriggerBindings |
| `controller_triggertemplate_count` | Gauge | | Number of TriggerTemplates |
| `controller_clustertriggerbinding_count` | Gauge | | Number of ClusterTriggerBindings |
| `controller_clusterinterceptor_count` | Gauge | | Number of ClusterInterceptors |

## EventListener Sink Metrics

The following metrics are exported by each EventListener sink pod on port
`9000`.

| Name | Type | Labels/Tags | Description |
|---|---|---|---|
| `eventlistener_event_received_total` | Counter | `status`=`succeeded`\|`failed` | Number of events received by the sink |
| `eventlistener_triggered_resources_total` | Counter | `kind`=&lt;resource kind&gt; | Number of resources created by triggers |
| `eventlistener_http_duration_seconds` | Histogram | | HTTP request duration in seconds |

> **Note:** Counter metrics include a `_total` suffix when exported via
> Prometheus. This is an OpenTelemetry/Prometheus convention.

## Configuring Metrics

Metrics are configured via the `config-observability-triggers` ConfigMap
in the `tekton-pipelines` namespace. By default, Prometheus export is
enabled on port `9000`.

See [config-observability.yaml](../config/config-observability.yaml) for
the full list of configuration options.

### Metrics protocol

The `metrics-protocol` key controls how metrics are exported:

| Value | Description |
|---|---|
| `prometheus` | Starts an HTTP server on port 9000 serving `/metrics` (default) |
| `grpc` | Exports via OTLP gRPC to the configured `metrics-endpoint` |
| `http/protobuf` | Exports via OTLP HTTP to the configured `metrics-endpoint` |
| `none` | Disables metrics export |

### Tracing protocol

The `tracing-protocol` key controls distributed tracing:

| Value | Description |
|---|---|
| `none` | Disables tracing (default) |
| `grpc` | Exports traces via OTLP gRPC to `tracing-endpoint` |
| `http/protobuf` | Exports traces via OTLP HTTP to `tracing-endpoint` |
| `stdout` | Prints traces to stdout (for debugging) |

### Example: Prometheus with OTLP tracing

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-observability-triggers
  namespace: tekton-pipelines
data:
  metrics-protocol: prometheus
  tracing-protocol: grpc
  tracing-endpoint: otel-collector.observability.svc.cluster.local:4317
  tracing-sampling-rate: "0.1"
```

### Verifying metrics

```bash
kubectl -n tekton-pipelines port-forward svc/tekton-triggers-controller 9000:9000
curl -s http://localhost:9000/metrics | grep -E "controller_eventlistener_count|controller_clusterinterceptor_count"
```

> **Note:** The previous OpenCensus-based configuration keys
> (`metrics.backend-destination`, `metrics.stackdriver-project-id`, etc.)
> are no longer supported. See the
> [migration guide](migration-guide-opencensus-to-opentelemetry.md) for
> details on upgrading from OpenCensus.
