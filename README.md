# Tekton Triggers
Triggers is a Kubernetes [Custom Resource Defintion](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) (CRD) controller that allows you to extract information from events payloads (a "trigger") to create Kubernetes resources. 

🚨 The contents of this repo are currently a WIP 🚨 We are working toward implementing
[this design](https://docs.google.com/document/d/1fngeNn3kGD4P_FTZjAnfERcEajS7zQhSEUaN7BYIlTw/edit#heading=h.iyqzt1brkg3o)
(visible to members of [the Tekton mailing list](https://github.com/tektoncd/community/blob/master/contact.md#mailing-list)).

## Background
[Tekton](https://github.com/tektoncd/pipeline) is a **Kubernetes-native**, continuous integration and delivery (CI/CD) framework that enables you to create containerized, composable, and configurable workloads declaratively through CRDs. Naturally, CI/CD events contain information that should:
- Identify the kind of event (Github Push, Gitlab Issue, Docker Hub Webhook, etc.)
- Be accessible from and map to particular pipelines (Take SHA from payload to use it in pipeline X)
- Deterministically trigger pipelines (Events/pipelines that trigger pipelines based on certain payload values)

The Tekton API enables functionality to be seperated from configuration (e.g. [Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md) vs [PipelineRuns](https://github.com/tektoncd/pipeline/blob/master/docs/pipelineruns.md)) such that steps can be reusable, but it does not provide a mechanism to generate the resources (notably, [PipelineRuns](https://github.com/tektoncd/pipeline/blob/master/docs/pipelineruns.md) and [PipelineResources](https://github.com/tektoncd/pipeline/blob/master/docs/resources.md#pipelineresources)) that encapsulate these configurations dynamically. Triggers extends the Tekton architecture with the following CRDs:
- `TriggerTemplate` - Templates resources to be created (e.g. Create PipelineResources and PipelineRun that uses them)
- `TriggerBinding` - Instantiates resources in TriggerTemplate using event fields
- `EventListener` - Wraps TriggerBinding(s) into an addressable endpoint (the event sink)


Using `tektoncd/triggers` in conjunction with `tektoncd/pipeline` enables you to easily create full-fledged CI/CD systems where the execution is defined **entirely** through Kubernetes resources. This repo draws inspiration from `Tekton`, but can used stand alone since `TriggerTemplates` can create any Kubernetes resource.

## Want to start using Tekton Triggers

Probably a bit early! In the meantime, consider
[ramping up on Tekton Pipelines](https://github.com/tektoncd/pipeline/tree/master/docs)

## Want to contribute

Hooray!

- See [CONTRIBUTING.md](CONTRIBUTING.md) for an overview of our processes
- See [DEVELOPMENT.md](DEVELOPMENT.md) for how to get started
- Look at our
  [good first issues](https://github.com/tektoncd/triggers/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
  and our
  [help wanted issues](https://github.com/tektoncd/triggers/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22)
