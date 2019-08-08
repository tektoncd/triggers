# EventListener
`EventListeners` encapsulate one or more `TriggerBindings` into an addressable endpoint, which is where webhooks/events are directed. When an `EventListener` is successfully created, a service is created that references a listener pod. This listener pod accepts the incoming events and does what has been specified in the corresponding `TriggerBindings`/`TriggerTemplates`.

<!-- FILE: examples/eventlisteners/eventlistener.yaml -->
```YAML
apiVersion: tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: listener
  namespace: tekton-pipelines
spec:
  triggerbindingrefs:
    - name: pipeline-binding
```
