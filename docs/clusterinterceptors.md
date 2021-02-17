<!--
---
linkTitle: "ClusterInterceptor"
weight: 9
---
-->
# ClusterInterceptor

A `ClusterInterceptor` is cluster scoped resource that registers a new Interceptor that 
can be invoked during the processing of a trigger to modify the behavior or payload of Triggers. The
custom resource describes how an EventListener can connect to a workload that
is running the interceptor business logic (and in the future what extra
paramters the interceptor accepts).

A `ClusterInterceptor` will consist of a custom resource of type ClusterInterceptor that defines it and a 
deployment/service combination that runs a HTTP server containing the actual interceptor processing logic. The 
interceptor can process the incoming request, decide to stop or continue processing, and add extra fields that are 
available to `TriggerBindings` and other interceptors in a chain under the `extensions` field.

**NOTE**: This doc describes a ClusterInterceptor and its interface. Please see the [Interceptors section](./eventlisteners.md#interceptors) 
in the EventListener doc for details on how to configure a Trigger to use an interceptor.

- [Interceptors](#interceptors)
  - [Syntax](#syntax)
    - [clientConfig](#clientConfig)
  - [Interceptor Services](#interceptor-services)

## Syntax

To define a configuration file for a `ClusterInterceptor` resource, you can specify
the following fields:

- Required:
  - [`apiVersion`][kubernetes-overview] - Specifies the API version, for example
    `triggers.tekton.dev/v1alpha1`.
  - [`kind`][kubernetes-overview] - Specifies the `ClusterInterceptor` resource
    object.
  - [`metadata`][kubernetes-overview] - Specifies data to uniquely identify the
    `ClusterInterceptor` resource object, for example a `name`.
  - [`spec`][kubernetes-overview] - Specifies the configuration information for
    your ClusterInterceptor resource object. The spec include:
    - [`clientConfig`] -  Specifies how a client (e.g. an EventListener) can communicate with the ClusterInterceptor.

### clientConfig

The `clientConfig` field describes how a client can communicate with an
interceptor. It can contain either the `url` field whose value is
a resolvable URL or it can contain a reference to a Kubernetes service where the ClusterInterceptor is running.
EventListeners will send forward requests to this service or URL.

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

## Interceptor Services

To be a valid ClusterInterceptor, the workload should satisfy the following:

- Be fronted by a regular Kubernetes v1 Service and serve HTTP
- Accept HTTP POST requests that contain a [`InterceptorRequest`](https://pkg.go.dev/github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1#InterceptorRequest) 
  JSON body
- Respond with a status code of 200 OK with [`InterceptorResponse`](https://pkg.go.dev/github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1#InterceptorResponse) 
  body
- Respond with a non 200 response only if something catastrophic went wrong during processing
