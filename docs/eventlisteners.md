# EventListener
`EventListeners` encapsulate one or more `TriggerBindings` into an addressable endpoint, which is where webhooks/events are directed. When an `EventListener` is successfully created, a service is created that references a listener pod. This listener pod accepts the incoming events and does what has been specified in the corresponding `TriggerBindings`/`TriggerTemplates`.

```YAML
apiVersion: tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: simple-listener
spec:
  triggerbindingrefs:
    - simple-pipeline-binding
```