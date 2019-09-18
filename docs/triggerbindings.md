# TriggerBindings
As per the name, `TriggerBinding`s bind against events/triggers.
`TriggerBinding`s enable you to capture fields within an event payload and store
them as parameters. The separation of `TriggerBinding`s from `TriggerTemplate`s
was deliberate to encourage reuse between them.

<!-- FILE: examples/triggerbindings/triggerbinding.yaml -->
```YAML
apiVersion: tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: pipeline-binding
spec:
  inputParams:
    - name: message
      description: The message to print
      default: This is the default message
  outputParams:
    - name: gitrevision
      value: $(event.head_commit.id)
    - name: gitrepositoryurl
      value: $(event.repository.url)
    - name: message
      value: $(inputParams.message)
```

`TriggerBinding`s are connected to `TriggerTemplate`s within an [`EventListener`](eventlisteners.md), which is where the pod is actually instantiated that "listens" for the respective events.

## Input Parameters
`TriggerBinding`s can declare input parameters that are supplied by an
`EventListener`. `inputParams` must have a `name`, and can have an optional
`description` and `default` value.

`inputParams` can be referenced in the `TriggerBinding` using the following
variable substitution syntax, where `<name>` is the name of the input parameter:
```YAML
$(inputParams.<name>)
```
`inputParams` can be referenced in the `outputParams` section of a
`TriggerBinding`. The purpose of `inputParams` is to make `TriggerBindings`
reusable.

## Output Parameters
`TriggerBinding`s can provide output params which are passed to a
`TriggerTemplate`. Each output parameter has a `name` and a `value`.
