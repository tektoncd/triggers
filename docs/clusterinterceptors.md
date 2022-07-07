<!--
---
linkTitle: "ClusterInterceptors"
weight: 5
---
-->
# `ClusterInterceptors`

Tekton Triggers ships with the `ClusterInterceptor` Custom Resource Definition (CRD), which allows you to implement a custom cluster-scoped Webhook-style `Interceptor`.

A `ClusterInterceptor` specifies an external Kubernetes v1 Service running custom business logic that receives the event payload from the
`EventListener` via an HTTP request and returns a processed version of the payload along with an HTTP 200 response. The `ClusterInterceptor` can also
halt processing if the event payload does not meet criteria you have configured as well as add extra fields that are accessible in the `EventListener's`
top-level `extensions` field to other [`Interceptors`](interceptors.md) and `ClusterInterceptors` chained with it and the associated `TriggerBinding`.

## Structure of a `ClusterInterceptor`

A `ClusterInterceptor` definition consists of the following fields:

- Required:
  - [`apiVersion`][kubernetes-overview] - specifies the target API version, for example `triggers.tekton.dev/v1alpha1`
  - [`kind`][kubernetes-overview] - specifies that this Kubernetes resource is a `ClusterInterceptor` object
  - [`metadata`][kubernetes-overview] - specifies data that uniquely identifies this `ClusterInterceptor` object, for example a `name`
  - [`spec`][kubernetes-overview] - specifies the configuration information for this `ClusterInterceptor` object, including:
    - [`clientConfig`] -  specifies how a client, such as an `EventListener` communicates with this `ClusterInterceptor` object

[kubernetes-overview]:
  https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields

## Configuring the client of the `ClusterInterceptor`

The `clientConfig` field specifies the client, such as an `EventListener` and how it communicates with the `ClusterInterceptor` to exchange
event payload and other data. You can configure this field in one of the following ways:

- Specify the `url` field and as its value a URL at which the corresponding Kubernetes service listens for incoming requests from this `ClusterInterceptor`
- Specify the `service` field and within it reference the corresponding Kubernetes service that's listening for incoming requests from this `ClusterInterceptor`

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

## Configuring a Kubernetes Service for the `ClusterInterceptor`

The Kubernetes object running the custom business logic for your `ClusterInterceptor` must meet the following criteria:

- Fronted by a regular Kubernetes v1 Service listening on an HTTP port (default port is 80)
- Accepts an HTTP `POST` request that contains an [`InterceptorRequest`](https://pkg.go.dev/github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1#InterceptorRequest) 
  as a JSON body
- Returns an HTTP 200 OK response that contains an [`InterceptorResponse`](https://pkg.go.dev/github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1#InterceptorResponse) 
  as a JSON body. If the trigger processing should continue, the interceptor should set the `continue` field in the response to `true`. If the processing should be stopped, the interceptor should set the `continue` field to `false` and also provide additional information detailing the error in the `status` field.
- Returns a response other than HTTP 200 OK only if payload processing halts due to a catastrophic failure. 

### Running ClusterInterceptor as HTTPS

Triggers now run clusterinterceptor as `https` server in order to support end to end secure connection and here is a [TEP](https://github.com/tektoncd/community/blob/main/teps/0102-https-connection-to-triggers-interceptor.md) which gives more detail about this support.

By default Triggers run all core interceptor (GitHub, GitLab, BitBucket, CEL) as `HTTPS`.

Triggers expose a new optional field `caBundle` as part of clusterinterceptor spec.

Example:
```yaml
spec:
  clientConfig:
    caBundle: <cert data>
    service:
      name: "my-interceptor-svc"
      namespace: "default"
      path: "/optional-path" # optional
      port: 8443
```

Triggers uses knative pkg to generate key, cert, cacert and fill caBundle for core interceptors (GitHub, GitLab, BitBucket, CEL).

Triggers now support writing custom interceptor for both `http` and `https`. Support of `http` for custom interceptor will be there for 1-2 releases, later it will be removed and only `https` will be supported. 
 
End user who write `https` custom interceptor need to pass `caBundle` as well as label
```yaml
  labels:
    server/type: https
```
to `ClusterInterceptor` in order to make secure connection with eventlistener.

Here is the reference for writing [https server for custom interceptor](https://github.com/tektoncd/triggers/blob/main/cmd/interceptors/main.go). 
