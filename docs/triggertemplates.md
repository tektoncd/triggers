# TriggerTemplates
A `TriggerTemplate` is a resource that can template resources.
`TriggerTemplates` have optional parameters that can be substituted **anywhere** within the resource template.
If resources do not have a name specified, it will default to value of the resource kind.
Further, all resources names will have a unique timestamp postfix to avoid naming conflicts.

<!-- FILE: examples/triggertemplates/triggertemplate.yaml -->
```YAML
apiVersion: tekton.dev/v1alpha1
kind: TriggerTemplate
metadata:
  name: pipeline-template
  namespace: tekton-pipelines
spec:
  params:
    - name: gitrevision
      description: The git revision
      default: master
    - name: gitrepositoryurl
      description: The git repository url
    - name: namespace
      description: The namespace to create the resources
  resourcetemplates:
    - apiVersion: tekton.dev/v1alpha1
      kind: PipelineResource
      metadata:
        name: git-source
        namespace: $(params.namespace)
        labels:
          triggertemplated: "true"
      spec:
        type: git
        params:
        - name: revision
          value: $(params.gitrevision)
        - name: url
          value: $(params.gitrepositoryurl)
    - apiVersion: tekton.dev/v1alpha1
      kind: PipelineRun
      metadata:
        name: simple-pipeline-run
        namespace: $(params.namespace)
        labels:
          triggertemplated: "true"
      spec:
        pipelineRef:
            name: simple-pipeline
        resources:
        - name: git-source
          resourceRef:
            name: git-source
```

Similar to [Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md),`TriggerTemplates` do not do any actual work, but instead act as the blueprint for what resources should be created.
Any parameters defined in the [`TriggerBinding`](triggerbindings.md) are passed into to the `TriggerTemplate` before resource creation.

To enable support for any resource type, the resource templates are internally resolved as byte blobs.
As a result, validation on these resources is only done at event processing time (rather than during `TriggerTemplate` creation).
