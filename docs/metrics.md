<!--
---
linkTitle: "Triggers Metrics"
weight: 1200
---
-->
# Tekton Triggers Controller Metrics

The following tekton triggers metrics are available at `controller-service` on port `9000`.

We expose several kinds of exporters, including Prometheus, Google Stackdriver, and many others. You can set them up using [observability configuration](../config/config-observability.yaml).

|  Name | Type | Description |
| ---------- | ----------- | ----------- |
| `controller_clusterinterceptor_count` | Gauge | Number of clusterinteceptor |
| `controller_eventlistener_count` | Gauge | Number of eventlistener |
| `controller_clustertriggerbinding_count` | Gauge | Number of clustertriggerbinding |
| `controller_triggerbinding_count` | Gauge | Number of triggerbinding |
| `controller_triggertemplate_count` | Gauge | Number of triggertemplate |

