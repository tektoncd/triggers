# EventListener
`EventListener`s connect `TriggerBinding`s to `TriggerTemplate`s and provide an
addressable endpoint, which is where webhooks/events are directed.

It also define an optional field called `validate` to validate event using a
predefined task. Learn more here: [validate-event](validate-event.md).

Further, it is at this level that the service account is connected, which
specifies what permissions the resources will be created (or at least attempted)
with.

When an `EventListener` is successfully created, a service is created that
references a listener pod. This listener pod accepts the incoming events and
does what has been specified in the corresponding
`TriggerBinding`s/`TriggerTemplate`s.

<!-- FILE: examples/eventlisteners/eventlistener.yaml -->
```YAML
apiVersion: tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: listener
spec:
  serviceAccountName: tekton-triggers-example-sa
  triggers:
    - name: foo-trig
      binding:
        name: pipeline-binding
      template:
        name: pipeline-template
      validate:
          taskRef:
           name: validate-github-event
          serviceAccountName: tekton-triggers-example-sa
          params:
          - name: Github-Secret
            value: githubsecret
          - name: Github-Secret-Key
            value: secretToken
      params:
      - name: message
        value: Hello from the Triggers EventListener!
```

## Parameters
`EventListener`s can provide `params` which are merged with the `TriggerBinding`
`params` and passed to the `TriggerTemplate`. Each parameter has a `name` and a
`value`.
