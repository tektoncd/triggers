# Tekton Triggers
Triggers enables users to map fields from an event payload into resource templates. Put another way, this allows events to both model and instantiate themselves as Kubernetes resources. In the case of `tektoncd/pipeline`, this makes it easy to encapsulate configuration into `PipelineRun`s and `PipelineResource`s. 

![TriggerFlow](../images/TriggerFlow.png)
# Learn more
See the following topics for more on each of the resources involved:
- [`TriggerTemplate`](triggertemplates.md)
- [`TriggerBinding`](triggerbindings.md)
- [`EventListener`](eventlisteners.md)
