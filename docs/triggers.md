<!--
---
linkTitle: "Trigger"
weight: 9
---
-->
# Triggers

A `Trigger` is resource that combines `TriggerTemplate`, `TriggerBindings` and `interceptors`. The `Trigger` is processed by EventListener which referenced it when it receives an incoming.

- [Triggers](#triggers)
  - [Syntax](#syntax)
    - [template](#template)

## Syntax

To define a configuration file for an `Trigger` resource, you can specify
the following fields:

- Required:
  - [`apiVersion`][kubernetes-overview] - Specifies the API version, for example
    `triggers.tekton.dev/v1alpha1`.
  - [`kind`][kubernetes-overview] - Specifies the `Trigger` resource
    object.
  - [`metadata`][kubernetes-overview] - Specifies data to uniquely identify the
    `Trigger` resource object, for example a `name`.
  - [`spec`][kubernetes-overview] - Specifies the configuration information for
    your Trigger resource object. The spec include:
    - [`bindings`] - (Optional) A list of bindings to use. Can either be a reference to existing `TriggerBinding` resources or embedded name/value pairs.
    - [`template`] - Either a reference to a TriggerTemplate object or an embedded TriggerTemplate spec.
    - [`interceptors`](./eventlisteners.md#interceptors) - (Optional) list of interceptors to use
    - [`serviceAccountName`] - (Optional) Specifies the ServiceAccount provided to EventListener by Trigger to create resources


<!-- FILE: examples/triggers/trigger.yaml -->
```YAML
apiVersion: triggers.tekton.dev/v1alpha1
kind: Trigger
metadata:
  name: trigger
spec:
  interceptors:
    - cel:
        filter: "header.match('X-GitHub-Event', 'pull_request')"
        overlays:
        - key: extensions.truncated_sha
          expression: "body.pull_request.head.sha.truncate(7)"
  bindings:
  - ref: pipeline-binding
  template:
    ref: pipeline-template
```

### template
The `template` field inside a `Trigger` can either point to an existing `TriggerTemplate` object (using `name`) or the template spec can be embedded inside the Trigger using the `spec` field:

```yaml
# Embedded Template Spec
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

