<!--
---
linkTitle: "EventListeners"
weight: 3
---
-->
# `EventListeners`

An `EventListener` is a Kubernetes object that listens for events at a specified port on your Kubernetes cluster. 
It exposes an addressable sink that receives incoming event and specifies one or more [`Triggers`](./triggers.md).
The sink is a Kubernetes service running the sink logic inside a dedicated Pod.

Each `Trigger`, in turn, allows you to specify one or more [`TriggerBindings`](./triggerbindings.md) that allow
you to extract fields and their values from event payloads, and one or more [`TriggerTemplates`](./triggertemplates.md)
that receive field values from the corresponding `TriggerBindings` and allow Tekton Triggers to instantiate resources
such as `TaskRuns` and `PipelineRuns` with that data. 

If you need to modify, filter, or validate the event payload data before passing it to a `TriggerBinding`, you can optionally specify one
or more [`Interceptors`](./interceptors.md).

- [Structure of an `EventListener`](#structure-of-an-eventlistener)
- [Specifying the Kubernetes service account](#specifiying-the-kubernetes-service-account)
- [Specifying `Triggers`](#specifying-triggers)
- [Specifying `Resources`](#specifying-resources)
  - [Specifying a `kubernetesResource` object](#specifying-a-kubernetesresource-object)
    - [Specifying `Replicas`](#specifying-replicas)
  - [Specifying a `CustomResource` object](#specifying-a-customresource-object)
    - [Contract for the `CustomResource` object](#contract-for-the-customresource-object)
- [Specifying `Interceptors`](#specifying-interceptors)
- [Constraining `EventListeners` to specific namespaces](#constraining-eventlisteners-to-specific-namespaces)
- [Labels in `EventListeners`](#labels-in-eventlisteners)
  - [Constraining `EventListeners` to specific labels](#constraining-eventlisteners-to-specific-labels)
- [Specifying `EventListener` timeouts](#specifying-eventlistener-timeouts)
- [Annotations in `EventListeners`](#annotations-in-eventlisteners)
- [Understanding `EventListener` response](#understanding-eventlistener-response)
- [TLS HTTPS support in `EventListeners`](#tls-https-support-in-eventlisteners)
- [Obtaining the status of deployed `EventListeners`](#obtaining-the-status-of-deployed-eventlisteners)
- [Configuring logging for `EventListeners`](#configuring-logging-for-eventlisteners)
- [Exposing an `EventListener` outside of the cluster](#exposing-an-eventlistener-outside-of-the-cluster)
  - [Exposing an `EventListener` using a Kubernetes `Ingress` object](#exposing-an-eventlistener-using-a-kubernetes-ingress-object)
  - [Exposing an `EventListener` using the NGINX Ingress Controller](#exposing-an-eventlistener-using-the-nginx-ingress-controller)
  - [Exposing an `EventListener` using OpenShift Route](#exposing-an-eventlistener-using-openshift-route)
- [Understanding the deployment of an `EventListener`](#understanding-the-deployment-of-an-eventlistener)
- [Deploying `EventListeners` in multi-tenant scenarios](#deploying-eventlisteners-in-multi-tenant-scenarios)
  - [Deploying each `EventListener` in its own namespace](#deploying-each-eventlistener-in-its-own-namespace)
  - [Deploying multiple `EventListeners` in the same namespace](#deploying-multiple-eventlisteners-in-the-same-namespace)


## Structure of an `EventListener`

An `EventListener` definition consists of the following fields:

- Required:
  - [`apiVersion`][kubernetes-overview] - specifies the target API version, for example `triggers.tekton.dev/v1alpha1`
  - [`kind`][kubernetes-overview] - specifies that this Kubernetes resource is an `EventListener` object
  - [`metadata`][kubernetes-overview] - specifies data that uniquely identifies this `EventListener` object, for example a `name`
  - [`spec`][kubernetes-overview] - specifies the configuration of your `EventListener`:
    - [`serviceAccountName`](#specifiying-the-kubernetes-service-account) - Specifies the `ServiceAccount` the `EventListener` will use to instantiate Tekton resources
- Optional:
  - [`triggers`](#specifying-triggers) - specifies a list of `Triggers` to execute upon event detection
  - [`resources`](#specifying-resources) - specifies the resources that will be available to the event listening service
  - [`namespaceSelector`](#constraining-eventlisteners-to-specific-namespaces) - specifies the namespace for the `EventListener`; this is where the `EventListener` looks for the 
    specified `Triggers` and stores the Tekton objects it instantiates upon event detection
  - [`labelSelector`](#constraining-eventlisteners-to-specific-labels) - specifies the labels for which your `EventListener` recognizes `Triggers` and instantiates the specified Tekton objects

[kubernetes-overview]:
  https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields

See our [Tekton Triggers examples](https://github.com/tektoncd/triggers/tree/master/examples) for ready-to-use example `EventListener` definitions.

## Specifying the Kubernetes service account

You must specify a Kubernetes service account in the `serviceAccountName` field that the `EventListener` will use to instantiate Tekton objects.

Tekton Trigger creates 2 clusterroles while installing with necessary permissions required for an eventlistener. You can directly create bindings for your serviceaccount with the clusterroles.
- A Kubernetes RoleBinding with `tekton-triggers-eventlistener-roles` clusterrole.
- A Kubernetes ClusterRoleBinding with `tekton-triggers-eventlistener-clusterroles` clusterrole.
  
  You can checkout an example [here](../examples/rbac.yaml).
- If you're using `namespaceSelectors` in your `EventListener`, you will have to create an additional `ClusterRoleBinding ` 
  with `tekton-triggers-eventlistener-roles` clusterrole.

## Specifying `Triggers`

You can optionally specify one or more `Triggers` that define the actions to take when the `EventListener` detects a qualifying event. You can specify *either* a reference to an
external `Trigger` object *or* reference/define the `TriggerBindings`, `TriggerTemplates`, and `Interceptors` in the `Trigger` definition. A `Trigger` definition 
specifies the following fields:

- `name` - (optional) a valid [Kubernetes name](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set) that uniquely identifies the `Trigger`
- `interceptors` - (optional) a list of [`Interceptors`](#specifying-interceptors) that will process event payload data before passing it to the associated `TriggerBinding`
- `bindings` - (optional) a list of `TriggerBindings` for this `Trigger`; you can either reference existing `TriggerBindings` or embed their definitions directly
- `template` - (optional) a list of `TriggerTemplates` for this `Trigger`; you can either reference existing `TriggerTemplates` or embed their definitions directly
- `triggerRef` - (optional) a reference to an external [`Trigger`](./triggers.md)

Below is an example `Trigger` definition that references the desired `TriggerBindings`, `TriggerTemplates`, and `Interceptors`:

```yaml
triggers:
  - name: trigger-1
    interceptors:
      - github:
          eventTypes: ["pull_request"]
    bindings:
      - ref: pipeline-binding # Reference to a TriggerBinding object
      - name: message # Embedded Binding
        value: Hello from the Triggers EventListener!
    template:
      ref: pipeline-template
```

Below is an example `Trigger` definition that specifies a reference to an external `Trigger` object:

```yaml
triggers:
    - triggerRef: trigger
```

Below is an example `Trigger` definition that embeds a `triggerTemplate` definition directly:

```yaml
triggers:
  - name: "my-trigger"
    template:
      spec: 
        params:
          - name: "my-param-name"
        resourceTemplates:
        - apiVersion: "tekton.dev/v1beta1"
          kind: TaskRun
          metadata:
            generateName: "pr-run-"
          spec:
            taskSpec:
              steps:
              - image: ubuntu
                script: echo "hello there"
```

Below is an example `Trigger` definition tailored to a multi-tenant scenario in which you may not
want all of your `Trigger` objects to have the same permissions as the `EventListener`. In such case,
you can specify a different service account at the `Trigger` level. This service account overrides
the service account specified in the `EventListener`.

```yaml
triggers:
  - name: trigger-1
    serviceAccountName: trigger-1-sa
    interceptors:
      - github:
          eventTypes: ["pull_request"]
    bindings:
      - ref: pipeline-binding
      - ref: message-binding
    template:
      ref: pipeline-template
``` 

You must update the `Role` assigned to the service account specified in the `EventListener` as shown below
to allow it to impersonate the service account specified in the `Trigger`:

```yaml
rules:
- apiGroups: [""]
  resources: ["serviceaccounts"]
  verbs: ["impersonate"]
```

## Specifying `Resources`

You can optionally customize the sink deployment for your `EventListener` using the `resources` field. It accepts the following types of objects:
- Kubernetes Resource using the `kubernetesResource` field
- Custom Resource objects via the `CustomResource` field

Legal values for the `PodSpec` and `Containers` sub-fields for both `kubernetesResource` and `CustomResource` fields are:
```
ServiceAccountName
NodeSelector
Tolerations
Volumes
Containers
```

Legal values for the `Containers` sub-field are:
```
Resources
VolumeMounts
Env
```

### Specifying a `kubernetesResource` object

Below is an example `resources:` field definition specifying a `kubernetesResource` object:

```yaml
spec:
  resources:
    kubernetesResource:
      serviceType: NodePort
      spec:
        template:
          metadata:
            labels:
              key: "value"
            annotations:
              key: "value"
          spec:
            serviceAccountName: tekton-triggers-github-sa
            nodeSelector:
              app: test
            tolerations:
            - key: key
              value: value
              operator: Equal
              effect: NoSchedule
```

#### Specifying `Replicas`

You can optionally use the `replicas` field to instruct Tekton Triggers to deploy more than one instance of your `EventListener` in individual Kubernetes Pods.
If you do not specify this value, the default number of instances (and thus, the number of respective Pods) per `EventListener` is 1. If you set a value for the `replicas` field
while creating or upgrading the `EventListener's` YAML file, that value overrides any value you set manually later as as well as a value set by any other deployment
mechanism, such as HPA.

### Specifying a `CustomResource` object

You can specify a Kubernetes Custom Resource object using the `CustomResource` field. This field has one sub-field, `runtime.RawExtension` that allows you to specify dynamic objects.

#### Contract for the `CustomResource` object

The `CustomResource` object must abide by the contract shown below.

Contract-mandated CRD structure for the `spec` field:

```spec
   spec:
     template:
       metadata:
       spec:
```

Contract-mandated CRD structure for the `status` field:
```status
type EventListenerStatus struct {
  duckv1beta1.Status `json:",inline"`

  // EventListener is addressable via the DNS address of the sink.
  duckv1alpha1.AddressStatus `json:",inline"`
}
```

**Note:** The CRD must follow the [WithPod{}](https://github.com/knative/pkg/blob/master/apis/duck/v1/podspec_types.go#L41) spec. 

Below is an example `resources:` field definition specifying a `CustomResource` object using a [Knative Service](https://knative.dev/docs/):

**Note:** This example assumes that [Knative is installed](https://github.com/tektoncd/community/blob/main/teps/0008-support-knative-service-for-triggers-eventlistener-pod.md#note) on your cluster.

```yaml
spec:
  resources:
    customResource:
      apiVersion: serving.knative.dev/v1
      kind: Service
#      metadata:
#        name: knativeservice # name is optional; if not specified, Triggers substitutes the EventListener's name with an "el-" prefix, for example: el-github-knative-listener
      spec:
        template:
          spec:
            serviceAccountName: tekton-triggers-example-sa
            containers:
            - resources:
                requests:
                  memory: "64Mi"
                  cpu: "250m"
                limits:
                  memory: "128Mi"
                  cpu: "500m"
```

## Specifying `Interceptors`

An `Interceptor` is a "catch-all" event processor for a specific platform that runs before the `TriggerBinding`. It allows you to perform payload filtering,
verification (using a secret), transformation, define and test trigger conditions, and implement other useful processing. Once the event data passes through
an `Interceptor`, it then goes to the `Trigger` before you pass the payload data to the `TriggerBinding`. You can also use an `Interceptor` to modify the
behavior of the associated `Trigger`.

For more information, see [`Interceptors`](./interceptors.md).

## Constraining `EventListeners` to specific namespaces

You can optionally specify a list of namespaces in which your `EventListener` will search for `Triggers` and instantiate the specified Tekton objects using the `namespaceSelector` field.

If you omit this field, your `EventListener` will only recognize `Triggers` specified in its definition or found under one or more specified target [labels](#specifying-target-labels).

Below is an example `namespaceSelector` field that configures the `EventListener` to use the `foo` and `bar` namespaces:

```yaml
  namespaceSelector:
    matchNames:
    - foo
    - bar
```

If you want your `EventListener` to recognize `Triggers` across your entire cluster, use a wildcard as the only namespace:

```yaml
  namespaceSelector:
    matchNames:
    - *
```

## Constraining `EventListeners` to specific labels

You can optionally specify the labels for which your `EventListener` recognizes `Triggers` and instantiates the specified Tekton objects using the `labelSelector` field.
This field uses the standard Kubernetes `labelSelector` mechanism and supports the `matchExpressions` sub-field. If you omit the `labelSelector` field, the `EventListener`
accepts all resource labels.

Below is an example `labelSelector` field definition that constrains your `EventListener` to only recognize `Triggers` within its own namespace that are labeled `foo=bar`:

```yaml
  labelSelector:
    matchLabels:
      foo: bar
```

Below is an example `labelSelector` field definition that uses the `matchExpression` sub-field to specify expressions that allow the `EventListener` to recognize `Triggers`
across all namespaces in the cluster:

```yaml
  namespaceSelector:
    matchNames:
    - *
  labelSelector:
    matchExpressions:
      - {key: environment, operator: In, values: [dev,stage]}
      - {key: trigger-phase, operator: NotIn, values: [testing]}
```

## Specifying `EventListener` timeouts

An `EventListener` times out if it cannot process an event request within a timeout specified in [controller.yaml](../config/controller.yaml). The timeouts are as follows:
- `-el-readtimeout`: Read timeout; default is 5 seconds.
- `-el-writetimeout`: Write timeout; default is 40 seconds.
- `-el-idletimeout`: Idle timeout; default is 120 seconds.
- `-el-timeouthandler`: Server route handler timeout; default is 30 seconds.

## Labels in `EventListeners`

By default, each `EventListener` automatically attaches the following labels to all resources it instantiates:

| Name                              | Description                                                 |
| --------------------------------- | ----------------------------------------------------------- |
| triggers.tekton.dev/eventlistener | Name of the `EventListener` that instantiated the resource. |
| triggers.tekton.dev/trigger       | Name of the `Trigger` that instantiated the resource.       |
| triggers.tekton.dev/eventid       | UID of the incoming event.                                  |

**Note:** Because they're used as labels, `EventListener` and `Trigger` names must conform to the [Kubernetes syntax and character set requirements](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set).

## Annotations in `EventListeners`

Tekton Triggers propagates all annotations that you include in your `EventListener` to the Kubernetes service and deployment created by that `EventListener`.
Keep in mind that annotations propagated from the `EventListener` override annotations already present in its Kubernetes service and deployment.

Below is an example load balancer protocol annotation in an `EventListener` definition that automatically propagates to the `EventListener's` service:

```
apiVersion: triggers.tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: eventlistener
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: https
```

## Understanding `EventListener` response

An `EventListener` responds with a `202 ACCEPTED` HTTP response when the `EventListener`
has been able to process the request and selected the appropriate triggers to process
based off the `EventListener` configuration. 

After detecting an event, the `EventListener` responds with the following message:

```json
{
  "eventListener": "listener",
  "namespace": "default",
  "eventListenerUID": "ea71a6e4-9531-43a1-94fe-6136515d938c",
  "eventID": "14a657c3-6816-45bf-b214-4afdaefc4ebd"
}
```

- `eventListenerUID` - [UID](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids) of the target EventListener.
- `eventID` - UID assigned to this event request

### Deprecated Fields

These fields are included in `EventListener` responses, but will be removed in a future release.

- `eventListener` - name of the target EventListener. Use `eventListenerUID` instead.
- `namespace` - namespace of the target EventListener. Use `eventListenerUID` instead.

## TLS HTTPS support in `EventListeners`

Tekton Triggers supports both HTTP and TLS-based HTTPS connections. To configure your `EventListener` for TLS,
add the `TLS_CERT` and `TLS_KEY` reserved environment variables using the `secretKeyRef` variable type, then
specify a `secret` containing the `cert` and `key` files. See [TEP-0027](https://github.com/tektoncd/community/blob/master/teps/0027-https-connection-to-triggers-eventlistener.md)
and our [TLS configuration example](../examples/eventlistener-tls-connection/README.md) for more information.

## Obtaining the status of deployed `EventListeners`

Use the following command to get a list of `EventListeners` deployed on your cluster along with their statuses:
```
kubectl get el
```

You will get a response similar to the following:
```
NAME                       ADDRESS                                                             AVAILABLE   REASON                     READY   REASON
tls-listener-interceptor   http://el-tls-listener-interceptor.default.svc.cluster.local        True        MinimumReplicasAvailable
```

Where for each returned line, the column values are, from left to right:
- `NAME` - name of the `EventListener`
- `ADDRESS` - IP address or URL of the `EventListener`
- `AVAILABLE` - readiness state of the associated `Deployment` and `Service`
- `REASON` - reason for the value displayed in the `AVAILABLE` column
- `READY` - readiness state of the Kubernetes Custom Resource object specified in the `EventListener`
- `REASON` - reason for the value displayed in the `READY` column

**Note:** The status messaging described above is being refactored. For more information, see [Issue 932](https://github.com/tektoncd/triggers/issues/932).

## Configuring logging for `EventListeners`

You can configure logging for your `EventListener` using the `config-logging-triggers` `ConfigMap` located in the `EventListener's` namespace.
Tekton Triggers automatically creates and populates this `ConfigMap` with default values described in [config-logging.yaml](../config/config-logging.yaml).

To access your `EventListener` logs, query your cluster for Pods whose `eventlistener` label matches the name of your `EventListener` object. For example:

```shell
kubectl get pods --selector eventlistener=my-eventlistener
```

## Configuring metrics for `EventListeners`

The following pipeline metrics are available on the `eventlistener` Service on port `9000`.

|  Name | Type | Labels/Tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| `eventlistener_triggered_resources` | Counter | `kind`=&lt;kind&gt; | experimental |
| `eventlistener_event_count` | Counter | `status`=&lt;status&gt; <br> | experimental |
| `eventlistener_http_duration_seconds_[bucket, sum, count]` | Histogram | `status`=&lt;status&gt; <br> | experimental |

Several kinds of exporters can be configured for an `EventListener`, including Prometheus, Google Stackdriver, and many others.
You can configure metrics using the [`config-observability-triggers` config map](../config/config-observability.yaml) in the `EventListener` namespaces.
There is a `config-observability-triggers` configmap in the `tekton-pipelines` namespace that can be configured for the operation of the Triggers
webhook and controller components.

See [the Knative documentation](https://github.com/knative/pkg/blob/main/metrics/README.md) for more information about available exporters and configuration values.

## Exposing an `EventListener` outside of the cluster

By default, `ClusterIP` services such as `EventListeners` are only accessible within the cluster on which they are running. 
You can expose them outside of the cluster in one of the following ways:
- Using a Kubernetes `Ingress` object
- Using the NGINX Ingress Controller
- Using OpenShift Route

### Exposing an `EventListener` using a Kubernetes `Ingress` object

Use the Tekton [`create-ingress`](./getting-started/create-ingress.yaml) task to configure an `Ingress` object using self-signed certificates.
If you are using a cloud-based Kubernetes solution such as Google Kubernetes Engine, the Kubernetes `Ingress` object will not work. Instead,
use the NGINX Ingress Controller as described in the next section.

### Exposing an `EventListener` using the NGINX Ingress Controller

Below are instructions for installing and configuring the NGINX Ingress Controller with Tekton Triggers on Google Kubernetes Cluster version `1.13.7-gke.24`.
For instructions on installing the NGINX Ingress Controller on other Kubernetes services as well as additional installation options for the example below,
see the [NGINX Ingress Controller Installation Guide](https://kubernetes.github.io/ingress-nginx/deploy/).

1. Install the NGINX Ingress Controller:
   ```sh
   kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v0.34.1/deploy/static/provider/cloud/deploy.yaml
   ```
   
2. Obtain the name of your `EventListener` service:
   ```sh
    kubectl get el <EVENTLISTENR_NAME> -o=jsonpath='{.status.configuration.generatedName}{"\n"}'
   ```
   
3. Instantiate the `Ingress` object:
   ```yaml
   apiVersion: extensions/v1beta1
   kind: Ingress
   metadata:
     name: ingress-resource
     annotations:
       kubernetes.io/ingress.class: nginx
       nginx.ingress.kubernetes.io/ssl-redirect: "false"
   spec:
     rules:
       - http:
           paths:
             - path: /
               backend:
                 serviceName: EVENT_LISTENER_SERVICE_NAME # REPLACE WITH YOUR SERVICE NAME FROM STEP 2
                servicePort: 8080
   ```

   See [NGINX Configuration](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/) for more information on configuring the NGINX Ingress Controller.

4. Obtain the IP address of the `Ingress` object:
   ```   
   kubectl get ingress ingress-resource
   ``` 

5. Test the configuragion with `curl` or set up a GitHub Webhook that sends events to it.

## Exposing an `EventListener` using Openshift Route

Below are instructions for configuring an OpenShift 4.2 cluster running API version `v1.14.6+32dc4a0`. For more information,
see [Route Configuration](https://docs.openshift.com/container-platform/4.2/networking/routes/route-configuration.html).

1. Obtain the name of your `EventListener` service:
   ```sh
    oc get el <EVENTLISTENR_NAME> -o=jsonpath='{.status.configuration.generatedName}'
   ```

2. Expose the `EventListener` service:
   ```sh
    oc expose svc/[el-listener] # REPLACE el-listener WITH YOUR SERVICE NAME FROM STEP 1
   ```

3. Obtain the IP address of the exposed route:
   ```sh
    oc get route el-listener  -o=jsonpath='{.spec.host}' # REPLACE el-listener WITH YOUR SERVICE NAME FROM STEP 1
   ```
   
4. Test the configuragion with `curl` or set up a GitHub Webhook that sends events to it.

## Understanding the deployment of an `EventListener`

Below is a high-level walkthrough through the deployment of an `EventListener` using a GitHub example provided by Tekton Triggers.

1. Instantiate the example `EventListener` on your cluster:
   ```bash
   kubectl create -f https://github.com/tektoncd/triggers/tree/master/examples/github
   ```
   
   Tekton Triggers creates a new `Deployment` and `Service` for the `EventListener`. using the `EventListener` definition,
   [`metadata.labels`](https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#labels), and pre-existing values
   such as container `Image`, `Name`, and `Port`. Tekton Triggers uses the `EventListener` name prefixed with `el-` to name the
   `Deployment` and `Service` when instantiating them. For exampl,e if the `EventListener` name is `foo`, the `Deployment` and
   `Service` names are named `el-foo`.

2. Use `kubectl` to verify the `Deployment` is running on your cluster:
   ```bash
   kubectl get deployment
   NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
   el-github-listener-interceptor   1/1     1            1           11s

3. Use `kubectl` to verify the `Service` is running on your cluster:
   ```
   kubectl get svc
   NAME                             TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
   el-github-listener-interceptor   ClusterIP   10.99.188.140   <none>        8080/TCP   52s
   ```

4. Obtain the URL on which the `EventListener` is listening for events:
   ```bash
   kubectl get eventlistener
   NAME                          ADDRESS                                                              AVAILABLE   REASON
   github-listener-interceptor   http://el-github-listener-interceptor.ptest.svc.cluster.local:8080   True        MinimumReplicasAvailable
   ```

   See our [GitHub `EventListener` example](https://github.com/tektoncd/triggers/blob/master/examples/github/README.md) to try instantiating an `EventListener` locally.

## Deploying `EventListeners` in multi-tenant scenarios

`EventListeners` are effectively Tekton clients that use HTTP to bypass the normal Kubernetes authentication
mechanism established through `kubeconfig` files and the `kubectl config` command tree. Because of this,
you must be conscious of your configuration decisions, such as:
- How to securely expose each `EventListener` to the outside of your cluster,
- How to securely control how each `EventListener` and its associated objects, such as [`Interceptors`](./interceptors.md),
  interact with data on your cluster.

At the minimum, each `EventListener` specifies its own Kubernetes Service account as explained earlier and
it acts on all events it receives with the permissions of that service account. If your business needs mandate
more granular permission control across the `Triggers` and `Interceptors` specified in your `EventListeners`,
you have the following options:
- Deploy each `EventListener` in its own namespace
- Deploy multiple `EventListeners` in the same namespace
- Specify a separate service account for each `Trigger`

### Deploying each `EventListener` in its own namespace

In this scenario, you create multiple `EventListeners` that in turn use a variety of `Triggers` and `Interceptors`, 
each `EventListener` gets its own namespace. This way, you can use a different service account for each namespace
and tailor the permissions of those accounts to the functionality of their corresponding `EventListeners`. Because
creating a namespace often instantiates the necessary service accounts based on pre-configured permissions, this
also simplifies the deployment process as you simply need to update the permissions associated with those accounts.

However, this approach has the following drawbacks:
- Namespaces with separately associated `Secrets` and `ServiceAccounts` can be the most expensive items in the
  Kubernetes `etcd` store; on large clusters, the capacity of the `etcd` store can become a concern.
- Since each `EventListener` requires its own HTTP port to listen for events, you must configure your network
  to allow access to each corresponding IP address and port combination unless you configure an ingress abstraction
  layer, such as the Kubernetes `Ingress` object, the NGINX Ingress Controller, or OpenShift Route.

### Deploying multiple `EventListeners` in the same namespace

In this scenario, you create multiple `EventListeners` in the same namespace. This will require customization of
the associated service account(s), secret(s), and RBAC, since the automatically generated defaults are not always
ideal, but you will not incur a significant `etcd` store cost as in the multiple namespace scenario. Network security
and configuration overhead concerns, however, still apply as described earlier. You can also achieve a similar result
by specifying a separate service account for each `Trigger` used across your `EventListener` pool at the cost of
increased administration overhead.
