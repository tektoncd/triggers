# EventListener
`EventListener`s connect `TriggerBinding`s to `TriggerTemplate`s and provide an addressable endpoint, which is where webhooks/events are directed.
Further, it is as this level that the service account is connected, which specifies what permissions the resources will be created (or at least attempted) with.
When an `EventListener` is successfully created, a service is created that references a listener pod. This listener pod accepts the incoming events and does what has been specified in the corresponding `TriggerBinding`s/`TriggerTemplate`s.

<!-- FILE: examples/eventlisteners/eventlistener.yaml -->
```YAML
apiVersion: tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: listener
spec:
  serviceAccountName: tekton-triggers-example-sa
  triggers:
    - binding:
        name: pipeline-binding
      template:
        name: pipeline-template
```
