<!--
---
linkTitle: "Trigger"
weight: 9
---
-->
# `Triggers`

A `Trigger` specifies what happens when the [`EventListener`](./eventlisteners.md) detects an event. A `Trigger` specifies a [`TriggerTemplate`](./triggertemplates.md),
a [`TriggerBinding`](./triggerbindings.md), and optionally an [`Interceptor`](./eventlisteners.md#interceptors).

## Structure of a `Trigger`

When creating a `Trigger` definition you must specify the required fields and can also specify any of the optional fields listed below:

- Required:
  - [`apiVersion`][kubernetes-overview] - Specifies the API version; for example `triggers.tekton.dev/v1alpha1`.
  - [`kind`][kubernetes-overview] - Specifies that this resource object is a `Trigger` object.
  - [`metadata`][kubernetes-overview] - Specifies metadata to uniquely identify this `Trigger` object; for example a `name`.
  - [`spec`][kubernetes-overview] - Specifies the configuration information for this Trigger object, including:
    - [`bindings`] - (Optional) Specifies a list of field bindings; each binding can either reference an existing `TriggerBinding` or embedded a `TriggerBinding`
                     definition using a `name`/`value` pair.
    - [`template`] - Specifies the corresponding `TriggerTemplate` either as a reference as an embedded `TriggerTemplate` definition.
    - [`interceptors`] - (Optional) specifies one or more `Interceptors` that will process the payload data before passing it to the `TriggerTemplate`.
    - [`serviceAccountName`] - (Optional) Specifies the `ServiceAccount` to supply to the `EventListener` to instantiate/execute the target resources.

Below is an example `Trigger` definition:

<!-- FILE: examples/triggers/trigger.yaml -->
```YAML
apiVersion: triggers.tekton.dev/v1alpha1
kind: Trigger
metadata:
  name: trigger
spec:
  interceptors:
    - ref:
        name: "cel"
      params:
        - name: "filter"
          value: "header.match('X-GitHub-Event', 'pull_request')"
        - name: "overlays"
          value:
            - key: extensions.truncated_sha
              expression: "body.pull_request.head.sha.truncate(7)"
  bindings:
  - ref: pipeline-binding
  template:
    ref: pipeline-template
```

Triggers can reference templates and bindings based on data in the event body, headers, or processed extensions using the `dynamicRef` field. This allows contextual information to be pulled in from the event to determine which binding or template should be populated. The `dynamicRef` field for `bindings` and `templates` can be populated in the same manner as triggertemplate params.

This can be useful when different pipelines require different resources or workspaces to run. `TriggerTemplate` objects by themselves must have a static configuration structure, even if individual strings are able to be populated.

These dynamic resolutions also allow for a fallback option if the dynamic resolution does not succeed. If the `dynamicRef` fields are not present in the event body, the Trigger processing will use the `ref` field if populated.

<!-- FILE: examples/triggers/trigger-with-event-reference.yaml -->
```YAML
apiVersion: triggers.tekton.dev/v1alpha1
kind: Trigger
metadata:
  name: trigger
spec:
  interceptors:
    - cel:
        overlays:
        - key: extensions.truncated_sha
          expression: "body.pull_request.head.sha.truncate(7)"
  bindings:
  - dynamicRef: pipeline-$(body.repository.owner.login)-binding
  template:
    dynamicRef: pipeline-$(header.X-GitHub-Event)-trigger
    ref: pipeline-default-trigger
```

## Specifying the corresponding `TriggerTemplate`

In the `template` field,  you can do one of the following:

* Use the `name` parameter to reference an external `TriggerTemplate` object, or

* Use the `spec` parameter to directly embed a `TriggerTemplate` definition.

For example:

```yaml
# Example: embedded TriggerTemplate definition
triggers:
  - name: "my-trigger"
    template:
      spec: 
        params:
          - name: "my-param-name"
        resourceTemplates:
        - apiVersion: "tekton.dev/v1beta1"
          kind: TaskRun
          metadata:
            generateName: "pr-run-"
          spec:
            taskSpec:
              steps:
              - image: ubuntu
                script: echo "hello there"
```

[kubernetes-overview]:
  https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields

