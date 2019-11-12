# EventListener

`EventListener`s connect `TriggerBinding`s to `TriggerTemplate`s and provide an
addressable endpoint, which is where webhooks/events are directed. This is also
where the service account is connected, which specifies what permissions the
resources will be created (or at least attempted) with. 
The service account must have the following role bound.

<!-- FILE: examples/role-resources/role.yaml -->
```YAML
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: tekton-triggers-example-minimal
rules:
# Permissions for every EventListener deployment to function
- apiGroups: ["tekton.dev"]
  resources: ["eventlisteners", "triggerbindings", "triggertemplates", "tasks", "taskruns"]
  verbs: ["get"]
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch"]
# Permissions to create resources in associated TriggerTemplates
- apiGroups: ["tekton.dev"]
  resources: ["pipelineruns", "pipelineresources", "taskruns"]
  verbs: ["create"]
```

Note that currently, JSON is the only accepted MIME type for events.

When an `EventListener` is successfully created, a service is created that
references a listener pod. This listener pod accepts the incoming events and
does what has been specified in the corresponding
`TriggerBinding`s/`TriggerTemplate`s. The created service is by default of type
`ClusterIP`; any other pods running in the same Kubernetes cluster can access
services' via their cluster DNS. For external services to connect to your
cluster (e.g. GitHub sending webhooks), check out the guide on [exposing eventlisteners](./exposing-eventlisteners.md)

When the `EventListener` is created in the different namespace from the trigger component, `config-logging-triggers` ConfigMap
is created for the logging configuration in the namespace when it doesn't exist.  The ConfigMap with the default configuration can be created
by applying [config-logging.yaml](../config/config-logging.yaml)

`EventListener` `spec.serviceType` can be set to `ClusterIP (default)` | `NodePort` | `LoadBalancer`
to configure the underlying `Service` resource to make it reachable externally.

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
      bindings:
      - name: pipeline-binding
      template:
        name: pipeline-template
```

