<!--
---
linkTitle: "Trigger"
weight: 9
---
-->
# Triggers

A `Trigger` is resource that combines `TriggerTemplate`, `TriggerBindings` and `interceptors`. The `Trigger` is processed by EventListener which referenced it when it receives an incoming.

- [Syntax](#syntax)

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
    - [`bindings`] -  A list of `TriggerBindings` reference to use or embedded TriggerBindingsSpecs to use
    - [`template`] -  The name of `TriggerTemplate` to use
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
    name: pipeline-template
```

[kubernetes-overview]:
  https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields

