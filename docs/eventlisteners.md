# EventListener
`EventListener`s connect `TriggerBinding`s to `TriggerTemplate`s and provide an
addressable endpoint, which is where webhooks/events are directed.

Further, it is at this level that the service account is connected, which
specifies what permissions the resources will be created (or at least attempted)
with.

When an `EventListener` is successfully created, a service is created that
references a listener pod. This listener pod accepts the incoming events and
does what has been specified in the corresponding
`TriggerBinding`s/`TriggerTemplate`s. The service created is a `ClusterIP` service 
which means any other pods running in the same Kubernetes cluster can access it via the service's 
cluster DNS. For external services to connect to your cluster (e.g. GitHub 
sending webhooks), check out the guide on [exposing eventlisteners](./exposing-eventlisteners.md) 

## Parameters
`EventListener`s can provide `params` which are merged with the `TriggerBinding`
`params` and passed to the `TriggerTemplate`. Each parameter has a `name` and a
`value`.

## Event Interceptors

Triggers within an `EventListener` can optionally specify an interceptor field
which contains an [`ObjectReference`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#objectreference-v1-core) to a Kubernetes Service. If an interceptor
is specified, the `EventListener` sink will forward incoming events to the
service referenced by the interceptor over HTTP. The service is expected to
process the event and return a response back. The status code of the response
determines if the processing is successful and the returned body is used as
the new event payload by the EventListener and passed on the `TriggerBinding`.
An interceptor has an optional header field with key-value pairs that will be
merged with event headers before being sent; [canonical](https://github.com/golang/go/blob/master/src/net/http/header.go#L214)
names must be specified.

#### Event Interceptor Services

To be an Event Interceptor, a Kubernetes object should:
* Be fronted by a regular Kubernetes v1 Service over port 80
* Accept JSON payloads over HTTP
* Return a HTTP 200 OK Status if the EventListener should continue processing
  the event
* Return a JSON body back. This will be used by the EventListener as the event
  payload for any further processing. If the interceptor does not need to modify
  the body, it can simply return the body that it received.

<!-- FILE: examples/eventlisteners/eventlistener-interceptor.yaml -->
```YAML
apiVersion: tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: listener-interceptor
spec:
  serviceAccountName: tekton-triggers-example-sa
  triggers:
    - name: foo-trig
      interceptor:
        header:
        - name: Foo-Trig-Header1
          value: string-value
        - name: Foo-Trig-Header2
          value:
          - array-val1
          - array-val2
        objectRef:
          kind: Service
          name: gh-validate
          apiVersion: v1
          namespace: default
      binding:
        name: pipeline-binding
      template:
        name: pipeline-template
      params:
      - name: message
        value: Hello from the Triggers EventListener!
```
