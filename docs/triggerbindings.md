# TriggerBindings
As per the name, `TriggerBinding`s bind against events/triggers.
`TriggerBinding`s enable you to capture fields within an event payload and store them as parameters.
The separation of `TriggerBinding`s from `TriggerTemplate`s was deliberate to encourage reuse between them.

<!-- FILE: examples/triggerbindings/triggerbinding.yaml -->
```YAML
apiVersion: tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: pipeline-binding
  namespace: tekton-pipelines
spec:
  params:
    - name: gitrevision
      value: $(event.head_commit.id)
    - name: gitrepositoryurl
      value: $(event.repository.url)
    - name: namespace
      value: tekton-pipelines
```

`TriggerBinding`s are connected to `TriggerTemplate`s within an [`EventListener`](eventlisteners.md), which is where the pod is actually instantiated that "listens" for the respective events.
