<!-- prettier-ignore start -->
<!--
---
title: "Overview of Tekton Triggers"
linkTitle: "Overview"
weight: 3
description: >
  Tekton Triggers
cascade:
  github_project_repo: https://github.com/tektoncd/triggers
---
-->

<!-- prettier-ignore end -->


# Overview of Tekton Triggers

Tekton Triggers is a Tekton component that allows you to detect and extract information from events from a variety of sources and deterministically instantiate
and execute [`TaskRuns`](https://github.com/tektoncd/pipeline/blob/master/docs/taskruns.md) and [`PipelineRuns`](https://github.com/tektoncd/pipeline/blob/master/docs/pipelineruns.md)
based on that information. Tekton Triggers can also pass information extracted from events directly to `TaskRuns` and `PipelineRuns`. You install Tekton Triggers on your Kubernetes
cluster as an extension to Tekton Pipelines.

## How does Triggers work?

Tekton Triggers consists of a controller service that runs on your Kubernetes cluster as well as the following Kubernetes Custom Resource Definitions (CRDs) that extend
the functionality of Tekton Pipelines to support events:

*  [`EventListener`](https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md) - listens for events at a specified port on your Kubernetes cluster.
   Specifies one or more `Triggers`.

*  [`Trigger`](https://github.com/tektoncd/triggers/blob/master/docs/triggers.md) - specifies what happens when the `EventListener` detects an event. A `Trigger` specifies
   a `TriggerTemplate`, a `TriggerBinding`, and optionally, an `Interceptor`.

*  [`TriggerTemplate`](https://github.com/tektoncd/triggers/blob/master/docs/triggertemplates.md) - specifies a template for the `TaskRun` or `PipelineRun` you want to
   instantiate and execute when EventListener detects an event.

*  [`TriggerBinding`](https://github.com/tektoncd/triggers/blob/master/docs/triggerbindings.md) - specifies the fields in the event payload from which you want to extract
   data as well as the fields in your `TaskRun` or `PipelineRun` to populate with the extracted values. In other words, it *binds* payload fields to `TaskRun` or
   `PipelineRun` fields.

*  [`ClusterTriggerBinding`](https://github.com/tektoncd/triggers/blob/master/docs/clustertriggerbindings.md) - a cluster-scoped version of the `TriggerBinding`,
   especially useful for reuse within your cluster.

*  [`Interceptor`](https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#interceptors) - a "catch-all" event processor for a specific platform that
   runs before the `TriggerBinding` enabling you to perform payload filtering, verification (using a secret), transformation, define and test trigger conditions, and other
   useful processing. Once the event data passes through an interceptor, it then goes to the `Trigger` before you pass the payload data to the `TriggerBinding`.

   **Note:** `Interceptors` are currently part of the `EventListener` API but are being converted to a standalone CRD in [PR 960](https://github.com/tektoncd/triggers/pull/960). 


## What can I do with Triggers?

As an example, you can implement the following CI/CD workflow with Triggers:

1. Triggers listens for a git commit or a git pull request event. When it detects one, it executes a unit test [`Pipeline`](https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md) on the committed code.

2. Triggers listens for a git push event indicating the test finished successfully. When it detects one, it validates the test's outcome and executes a `Pipeline` that builds the tested code.

3. When the associated `PipelineRun` completes execution, Triggers checks the outcome of the build, and if it's successful, executes a [`Task`](https://github.com/tektoncd/pipeline/blob/master/docs/tasks.md) that uploads the build artifacts to the Docker registry of your choice.

4. Finally, the Docker registry sends an event to [Pub/Sub](https://cloud.google.com/pubsub/docs/overview), which triggers a `Pipeline` that pushes the build artifacts to a staging environment.


## Further Reading

To get started with Tekton Triggers, see the following:

*   [Setting Up Tekton Triggers](https://github.com/tektoncd/triggers/blob/master/docs/install.md)
*   [Getting Started with Tekton Triggers](https://github.com/tektoncd/triggers/blob/master/docs/getting-started/README.md)
*   [Tekton Triggers code examples](https://github.com/tektoncd/triggers/tree/master/examples)
