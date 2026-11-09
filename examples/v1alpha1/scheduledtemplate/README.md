# ScheduledTemplate

[`ScheduledTemplate`](../../../docs/scheduledtemplates.md) (TEP-0128) schedules creation of
Tekton resources (for example `PipelineRun` / `TaskRun`) using the same templating model as
`TriggerTemplate`, without wiring a CronJob to an EventListener.

## Prerequisites

1. Tekton Pipelines and Triggers are installed (this example needs a controller build that
   includes the ScheduledTemplate reconciler and `cronrunner` image).
2. A Pipeline named `simple-pipeline` exists in the namespace (or edit the example).
3. Apply RBAC so the runner ServiceAccount can create runs and read templates:

```bash
kubectl apply -f rbac.yaml
kubectl apply -f scheduledtemplate.yaml
```

## What gets created

The Triggers controller reconciles the `ScheduledTemplate` into an owned Kubernetes
`CronJob` named `st-<scheduledtemplate-name>`. On each schedule, the CronJob runs the
`cronrunner` image, which resolves the template (inline or `ref`) and creates the resources.

## CloudEvents

Set `spec.cloudEventSink` to an `http` or `https` URL to receive CloudEvents after each fire:

| Outcome | Type |
|---------|------|
| Success | `dev.tekton.event.triggers.scheduledtemplate.successful.v1` |
| Failure | `dev.tekton.event.triggers.scheduledtemplate.failed.v1` |

## Related

The older [cron example](../cron/) posts to an EventListener. Prefer `ScheduledTemplate` when
you only need scheduled resource creation from templates.
