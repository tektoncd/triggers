<!--
---
linkTitle: "ScheduledTemplates"
weight: 350
---
-->
# ScheduledTemplates

`ScheduledTemplate` (TEP-0128) creates Tekton (or other) resources on a cron schedule using
the same parameter and resource-template model as [`TriggerTemplate`](triggertemplates.md).

Unlike the [cron + EventListener example](../examples/v1alpha1/cron/), you do not maintain a
hand-written CronJob that POSTs to an EventListener. Applying a `ScheduledTemplate` is enough:
the Triggers controller owns a Kubernetes `CronJob` that runs the Triggers `cronrunner` on
schedule.

## Spec overview

| Field | Description |
|-------|-------------|
| `schedule` | Standard Kubernetes CronJob schedule (five fields). |
| `serviceAccountName` | ServiceAccount for the CronJob pod (defaults to `default`). Must be allowed to create the resources in the templates and to get the ScheduledTemplate / TriggerTemplate. |
| `ref` | Name of a namespaced `TriggerTemplate`. Mutually exclusive with inline `params` / `resourcetemplates`. |
| `params` / `resourcetemplates` | Inline TriggerTemplate-style templates (same as embedding `TriggerTemplateSpec`). |
| `cloudEventSink` | Optional `http`/`https` URL for CloudEvents after each fire. |

There are no TriggerBindings in v1alpha1: params come from template defaults (and any values
you bake into the template). Event payload substitution is not used on the schedule path.

## Status

| Field | Description |
|-------|-------------|
| `conditions` | Includes `CronJob` existence and overall `Ready`. |
| `cronJobName` | Name of the owned CronJob (`st-<name>`, truncated to Kubernetes CronJob name limits). |
| `observedGeneration` | Last reconciled generation. |
| `lastScheduleTime` / `lastSuccessfulTime` / `active` | Propagated from the owned CronJob when available. |

Created resources are labeled with:

- `triggers.tekton.dev/scheduledtemplate=<name>`
- `triggers.tekton.dev/triggers-eventid=<execution-id>` (per fire)

## CloudEvents

When `spec.cloudEventSink` is set, the runner sends a CloudEvent after each attempt:

| Type | When |
|------|------|
| `dev.tekton.event.triggers.scheduledtemplate.successful.v1` | All resources created |
| `dev.tekton.event.triggers.scheduledtemplate.failed.v1` | Resolve or create failed |

Payload fields include `scheduledTemplate`, `namespace`, `executionID`, and on failure `error`.

CloudEvent delivery failures are logged and do not change the Job exit code after a successful create.

## RBAC

The Triggers controller needs permission to manage `batch` `cronjobs` / `jobs` (included in
`tekton-triggers-admin`).

The CronJob pod ServiceAccount needs at least:

- `create` on the resource kinds in your templates (for example `pipelineruns`, `taskruns`)
- `get`/`list`/`watch` on `scheduledtemplates` and `triggertemplates`

See [`examples/v1alpha1/scheduledtemplate/`](../examples/v1alpha1/scheduledtemplate/).

## Example

```yaml
apiVersion: triggers.tekton.dev/v1alpha1
kind: ScheduledTemplate
metadata:
  name: nightly
spec:
  schedule: "0 0 * * *"
  serviceAccountName: tekton-triggers-scheduledtemplate-sa
  params:
    - name: git-revision
      default: main
  resourcetemplates:
    - apiVersion: tekton.dev/v1
      kind: PipelineRun
      metadata:
        generateName: nightly-
      spec:
        pipelineRef:
          name: build-pipeline
        params:
          - name: git-revision
            value: $(tt.params.git-revision)
```

Or reference an existing TriggerTemplate:

```yaml
spec:
  schedule: "0 * * * *"
  ref: my-trigger-template
  serviceAccountName: tekton-triggers-scheduledtemplate-sa
```

## Release notes (alpha)

- **Adds** `ScheduledTemplate` controller that owns a CronJob and creates resources on schedule via `cronrunner`.
- **Adds** optional `spec.serviceAccountName` (default `default`) and status fields for CronJob observation.
- **Adds** optional `spec.cloudEventSink` notifications for successful/failed fires.
