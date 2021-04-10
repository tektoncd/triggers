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
top-level `extensions` field to other `Interceptors` and `ClusterInterceptors` chained with it and the associated `TriggerBinding`.

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
  as a JSON body
- Returns a response other than HTTP 200 OK only if payload processing halts due to a catastrophic failure

## Interceptor SDK

The interceptor sdk allows you to easily create new interceptors for use with Tekton. The main SDK
function can be found in [pkg/interceptors/sharedmain.go](pkg/interceptors/sharedmain.go). Invoke the
function `sdk.InterceptorMainWithConfig` passing in a map with a key of the interceptor HTTP request
path and the value as an object that implements the `InterceptorInterface`.

The `Process` function of the `InterceptorInterface` is where you should implement the logic around
your interceptor HTTP request execution. The second function `InitializeContext` passes a context
that can be used to retrieve cancellation events from the top level HTTP server as well
as retrieving the `*zap.SugaredLogger` and `Kubernetes.Interface` objects. To retrieve these from the
context, use the following methods:

```
import (
  "knative.dev/pkg/logging"
  "github.com/tektoncd/triggers/pkg/interceptors"
)

func (w *Interceptor) InitializeContext(ctx context.Context) {
	w.Logger = logging.FromContext(ctx)
	w.KubeClientSet = interceptors.GetKubeClient(ctx)
}
```

If any of these functions are added to the resource type, the interceptor SDK will pass
the values from the main interceptor initialization to the downstream resource. To see
how these values are configured, see [`pkg/interceptors/sharedmain.go`](pkg/interceptors/sharedmain.go).

To confirm that your interceptor appropriate implements the `InterceptorInterface`, at the top of your
interceptor file, you can add the following:

```
import (
  	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

var _ triggersv1.InterceptorInterface = (*Interceptor)(nil)
```

This is done at the top of each of the core interceptors as an example.

