<!--
---
linkTitle: "Trigger"
weight: 9
---
-->
# `Triggers`

A `Trigger` specifies what happens when the [`EventListener`](./eventlisteners.md) detects an event. A `Trigger` specifies a [`TriggerTemplate`](./triggertemplates.md),
a [`TriggerBinding`](./triggerbindings.md), and optionally an [`Interceptor`](./interceptors.md).

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
    - `ref` - a reference to a [`ClusterInterceptor`](./clusterinterceptors.md) or [`Interceptor`](./namespacedinterceptors.md) object with the following fields:
      - `name` - the name of the referenced `ClusterInterceptor`
      - `kind` - (Optional) specifies that whether the referenced Kubernetes object is a `ClusterInterceptor` object or `NamespacedInterceptor`. Default value is `ClusterInterceptor`
    - [`serviceAccountName`] - (Optional) Specifies the `ServiceAccount` to supply to the `EventListener` to instantiate/execute the target resources.

Below is an example `Trigger` definition:

<!-- FILE: examples/v1beta1/trigger-ref/trigger.yaml -->
```YAML
apiVersion: triggers.tekton.dev/v1beta1
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
        resourcetemplates:
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

