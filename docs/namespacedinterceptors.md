<!--
---
linkTitle: "Namespaced Interceptors"
weight: 5
---
-->
# `Interceptors`

Tekton Triggers ships with the `Interceptor` Custom Resource Definition (CRD), which allows you to implement a custom namespaced-scoped Webhook-style `Interceptor`.

A `Interceptor` specifies an external Kubernetes v1 Service running custom business logic that receives the event payload from the
`EventListener` via an HTTP request and returns a processed version of the payload along with an HTTP 200 response. The `Interceptor` can also
halt processing if the event payload does not meet criteria you have configured as well as add extra fields that are accessible in the `EventListener's`
top-level `extensions` field to other [`Interceptors`](interceptors.md) and `Interceptors` chained with it and the associated `TriggerBinding`.

## Structure of a `Interceptor`

A `Interceptor` definition consists of the following fields:

- Required:
  - [`apiVersion`][kubernetes-overview] - specifies the target API version, for example `triggers.tekton.dev/v1alpha1`
  - [`kind`][kubernetes-overview] - specifies that this Kubernetes resource is a `Interceptor` object
  - [`metadata`][kubernetes-overview] - specifies data that uniquely identifies this `Interceptor` object, for example a `name`
  - [`spec`][kubernetes-overview] - specifies the configuration information for this `Interceptor` object, including:
    - [`clientConfig`] -  specifies how a client, such as an `EventListener` communicates with this `Interceptor` object

[kubernetes-overview]:
  https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields

## Configuring the client of the `Interceptor`

The `clientConfig` field specifies the client, such as an `EventListener` and how it communicates with the `Interceptor` to exchange
event payload and other data. You can configure this field in one of the following ways:

- Specify the `url` field and as its value a URL at which the corresponding Kubernetes service listens for incoming requests from this `Interceptor`
- Specify the `service` field and within it reference the corresponding Kubernetes service that's listening for incoming requests from this `Interceptor`

For example:

```yaml
spec:
  clientConfig:
    url: "http://interceptor-svc.default.svc/"
---
spec:
  clientConfig:
    service:
      name: "my-interceptor-svc"
      namespace: "default"
      path: "/optional-path" # optional
      port: 8081 # defaults to 80
```

## Configuring a Kubernetes Service for the `Interceptor`

The Kubernetes object running the custom business logic for your `Interceptor` must meet the following criteria:

- Fronted by a regular Kubernetes v1 Service listening on an HTTP port (default port is 80)
- Accepts an HTTP `POST` request that contains an [`InterceptorRequest`](https://pkg.go.dev/github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1#InterceptorRequest) 
  as a JSON body
- Returns an HTTP 200 OK response that contains an [`InterceptorResponse`](https://pkg.go.dev/github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1#InterceptorResponse) 
  as a JSON body. If the trigger processing should continue, the interceptor should set the `continue` field in the response to `true`. If the processing should be stopped, the interceptor should set the `continue` field to `false` and also provide additional information detailing the error in the `status` field.
- Returns a response other than HTTP 200 OK only if payload processing halts due to a catastrophic failure. 

### Running Interceptor as HTTPS

Triggers support writing custom interceptor for both `http` and `https`. Support of `http` for custom interceptor will be removed in future and only `https` will be supported.

End user who write `https` custom interceptor need to pass `caBundle` as well as label
```yaml
  labels:
    server/type: https
```
to `Interceptor` in order to make secure connection with eventlistener.

Here is the reference for writing [https server for custom interceptor](https://github.com/tektoncd/triggers/blob/main/cmd/interceptors/main.go). 
