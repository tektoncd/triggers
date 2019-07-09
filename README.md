# Tekton Triggers
Triggers is a Kubernetes [Custom Resource Defintion](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) (CRD) controller that allows you to extract information from events payloads (a "trigger") to create Kubernetes resources. 

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
