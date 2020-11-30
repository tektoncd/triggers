<!--
---
linkTitle: "Event Listeners"
weight: 5
---
-->
# EventListener

EventListener is a Kubernetes custom resource that allows users a declarative
way to process incoming HTTP based events with JSON payloads. EventListeners
expose an addressable "Sink" to which incoming events are directed. Users can
declare [TriggerBindings](./triggerbindings.md) to extract fields from events,
and apply them to [TriggerTemplates](./triggertemplates.md) in order to create
Tekton resources. In addition, EventListeners allow lightweight event processing
using [Event Interceptors](#Interceptors).

- [EventListener](#eventlistener)
  - [Syntax](#syntax)
    - [ServiceAccountName](#serviceaccountname)
    - [Triggers](#triggers)
    - [ServiceType](#servicetype)
    - [Replicas](#replicas)
    - [PodTemplate](#podtemplate)
    - [Resources](#resources)
    - [Logging](#logging)
    - [NamespaceSelector](#namespaceSelector)
  - [Labels](#labels)
  - [Annotations](#annotations)
  - [Interceptors](#interceptors)
    - [Webhook Interceptors](#webhook-interceptors)
      - [Event Interceptor Services](#event-interceptor-services)
    - [GitHub Interceptors](#github-interceptors)
    - [GitLab Interceptors](#gitlab-interceptors)
    - [Bitbucket Interceptors](#bitbucket-interceptors)
    - [CEL Interceptors](#cel-interceptors)
      - [Overlays](#overlays)
  - [EventListener Response](#eventlistener-response)
  - [How does the EventListener work?](#how-does-the-eventlistener-work)
  - [Examples](#examples)
  - [Response Timeout](#response-timeout)
  - [Multi-Tenant Concerns](#multi-tenant-concerns)
    - [Multiple EventListeners (One EventListener Per Namespace)](#multiple-eventlisteners-one-eventlistener-per-namespace)
    - [Multiple EventListeners (Multiple EventListeners per Namespace)](#multiple-eventlisteners-multiple-eventlisteners-per-namespace)
    - [ServiceAccount per EventListenerTrigger](#serviceaccount-per-eventlistenertrigger)
  - [EventListener Secure Connection](#eventlistener-secure-connection)
    - [Prerequisites](#prerequisites)

## Syntax

To define a configuration file for an `EventListener` resource, you can specify
the following fields:

- Required:
  - [`apiVersion`][kubernetes-overview] - Specifies the API version, for example
    `triggers.tekton.dev/v1alpha1`.
  - [`kind`][kubernetes-overview] - Specifies the `EventListener` resource
    object.
  - [`metadata`][kubernetes-overview] - Specifies data to uniquely identify the
    `EventListener` resource object, for example a `name`.
  - [`spec`][kubernetes-overview] - Specifies the configuration information for
    your EventListener resource object. In order for an EventListener to do
    anything, the spec must include:
    - [`triggers`](#triggers) - Specifies a list of Triggers to run
    - [`serviceAccountName`](#serviceAccountName) - Specifies the ServiceAccount
      that the EventListener uses to create resources
- Optional:
  - [`serviceType`](#serviceType) - Specifies what type of service the sink pod
    is exposed as
  - [`replicas`](#replicas) - Specifies the number of EventListener pods
  - [`podTemplate`](#podTemplate) - Specifies the PodTemplate
    for your EventListener pod
  - [`resources`](#resources) - Specifies the Kubernetes Resource information
    for your EventListener pod
  - [`namespaceSelector`](#namespaceSelector) - Specifies the namespaces where
    EventListener can fetch triggers from and create Tekton resources.

[kubernetes-overview]:
  https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields

### ServiceAccountName

The `serviceAccountName` field is required. The ServiceAccount that the
EventListener sink uses to create the Tekton resources. 
The ServiceAccount needs a Role that with "get", "list", and "watch" verbs for each Triggers resource as well as  a ClusterRole with read access to ClusterTriggerBindings. In addition, it needs to have "create"
permissions on the Pipeline resources it needs to create. See a working example at [../examples/rbac.yaml](../examples/rbac.yaml).

If your EventListener is using `namespaceSelectors`, the ServiceAccount will require a Cluster role to have read permissions for all Triggers resources across the cluster.

### Triggers

The `triggers` field is required. Each EventListener can consist of one or more
`triggers`. A Trigger consists of:

- `name` - (Optional) a valid
  [Kubernetes name](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set)
- [`interceptors`](#interceptors) - (Optional) list of interceptors to use
- `bindings` - (Optional) A list of bindings to use. Can either be a reference to existing `TriggerBinding` resources or embedded name/value pairs.
- `template` - (Optional) Either a reference to a TriggerTemplate object or an embedded TriggerTemplate spec.
- `triggerRef` - (Optional) Reference to the [`Trigger`](./triggers.md).

A `trigger` field must either have a `template` (along with needed `bindings` and `interceptors`) or a reference to another Trigger using `triggerRef`.

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

Or with only `triggerRef`:
```yaml
triggers:
    - triggerRef: trigger
```

Or with an embedded `triggerTemplate` spec:
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

Also, to support multi-tenant styled scenarios, where an administrator may not want all triggers to have
the same permissions as the `EventListener`, a service account can optionally be set at the trigger level
and used if present in place of the `EventListener` service account when creating resources:

```yaml
triggers:
  - name: trigger-1
    serviceAccount: 
      name: trigger-1-sa
      namespace: event-listener-namespace
    interceptors:
      - github:
          eventTypes: ["pull_request"]
    bindings:
      - ref: pipeline-binding
      - ref: message-binding
    template:
      ref: pipeline-template
``` 

An update to the `Role` assigned to the EventListener's SeviceAccount is needed to allow it to impersonate
the ServiceAccount specified for the trigger.

```yaml
rules:
- apiGroups: [""]
  resources: ["serviceaccounts"]
  verbs: ["impersonate"]
```

### ServiceType

The `serviceType` field is optional. EventListener sinks are exposed via
[Kubernetes Services](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types).
By default, the serviceType is `ClusterIP` which means any pods running in the
same Kubernetes cluster can access services' via their cluster DNS. Other valid
values are `NodePort` and `LoadBalancer`. Check the
[Kubernetes Service types](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types)
documentations for details.

For external services to connect to your cluster (e.g. GitHub sending webhooks),
check out the guide on [exposing EventListeners](./exposing-eventlisteners.md).

### Replicas

The `replicas` field is optional. By default, the number of replicas of EventListener is 1.
If you want to deploy more than one pod, you can specify the number to `replicas` field.

**Note:** If user sets `replicas` field while creating/updating eventlistener yaml then it won't respects replicas values edited by user manually on deployment or through any other mechanism (ex: HPA).

### PodTemplate

The `podTemplate` field is optional. A PodTemplate is specifications for 
creating EventListener pod. A PodTemplate consists of:
- `tolerations` - list of toleration which allows pods to schedule onto the nodes with matching taints.
This is needed only if you want to schedule EventListener pod to a tainted node.
- `nodeSelector` - key-value labels the node has which an EventListener pod should be scheduled on. 

```yaml
spec:
  podTemplate:
    nodeSelector:
      app: test
    tolerations:
    - key: key
      value: value
      operator: Equal
      effect: NoSchedule
```

### Resources

The `resources` field is optional.
Resource field helps to provide Kubernetes or custom resource information.

For more info on the design refer [TEP-0008](https://github.com/tektoncd/community/blob/master/teps/0008-support-knative-service-for-triggers-eventlistener-pod.md)

Right now the `resources` field is optional in order to support backward compatibility with original behavior of `podTemplate`, `serviceType` and `serviceAccountName` fieds.
In the future, we plan to deprecate `serviceAccountName`, `serviceType` and `podTemplate` from the EventListener spec in favor of the `resources` field.

For now `resources` has support for `kubernetesResource` but later it will have a support for Custom CRD`(ex: Knative Service)` as `customResource`

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

With the help of `kubernetesResource` user can specify [PodTemplateSpec](https://github.com/kubernetes/api/blob/master/core/v1/types.go#L3704).

Right now the allowed values as part of `podSpec` are
```text
ServiceAccountName
NodeSelector
Tolerations
Volumes
Containers
- Resources
- VolumeMounts
- Env
```

### Logging

EventListener sinks are exposed as Kubernetes services that are backed by a Pod
running the sink logic. The logging configuration can be controlled via the
`config-logging-triggers` ConfigMap present in the namespace that the
EventListener was created in. This ConfigMap is automatically created and
contains the default values defined in
[config-logging.yaml](../config/config-logging.yaml).

To access logs for the EventListener sink, you can query for pods with the
`eventlistener` label set to the name of your EventListener resource:

```shell
kubectl get pods --selector eventlistener=my-eventlistener
```

### NamespaceSelector
The `namespaceSelector` field is optional.
This field determines the namespaces where EventListener can search for triggers and
create Tekton resources. If this field isn't provided, EventListener will only serve Triggers from its
own namespace.

Snippet below will function in foo and bar namespaces.
```yaml
  namespaceSelector:
    matchNames:
    - foo
    - bar
```

If EventListener is required to listen to serve the whole cluster, then below snippet
can be used where we only provide single argument for `matchNames` as `*`.
```yaml
  namespaceSelector:
    matchNames:
    - *
```


## Labels

By default, EventListeners will attach the following labels automatically to all
resources it creates:

| Name                     | Description                                            |
| ------------------------ | ------------------------------------------------------ |
| triggers.tekton.dev/eventlistener | Name of the EventListener that generated the resource. |
| triggers.tekton.dev/trigger       | Name of the Trigger that generated the resource.       |
| triggers.tekton.dev/eventid       | UID of the incoming event.                             |

Since the EventListener name and Trigger name are used as label values, they
must adhere to the
[Kubernetes syntax and character set requirements](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set)
for label values.

## Annotations

All the annotations provided in Eventlistener will be further propagated to the service
and deployment created by that Eventlistener.

Example: You may need to add some annotation to the service like you need to annotation
for TLS support to a LoadBalancer service on AWS, you can specify that annotation in 
Eventlistener and it will be available to the service created by EventListener.

```
apiVersion: triggers.tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: eventlistener
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: https
```

**Note**: If there are any annotations attached to the service or the deployment, they will get overwritten 
by the annotations available in the eventlistener.

## Interceptors

Triggers within an `EventListener` can optionally specify interceptors, to
modify the behavior or payload of Triggers.

Event Interceptors can take several different forms today:

- [Webhook Interceptors](#Webhook-Interceptors)
- [GitHub Interceptors](#GitHub-Interceptors)
- [GitLab Interceptors](#GitLab-Interceptors)
- [Bitbucket Interceptors](#Bitbucket-Interceptors)
- [CEL Interceptors](#CEL-Interceptors)

### Webhook Interceptors

Webhook Interceptors allow users to configure an external k8s object which
contains business logic. These are currently specified under the `Webhook`
field, which contains an
[`ObjectReference`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#objectreference-v1-core)
to a Kubernetes Service. If a Webhook Interceptor is specified, the
`EventListener` sink will forward incoming events to the service referenced by
the Interceptor over HTTP. The service is expected to process the event and
return a response back. The status code of the response determines if the
processing is successful - a 200 response means the Interceptor was successful
and that processing should continue, any other status code will halt Trigger
processing. The returned request (body and headers) is used as the new event
payload by the EventListener and passed on the `TriggerBinding`. An Interceptor
has an optional header field with key-value pairs that will be merged with event
headers before being sent;
[canonical](https://golang.org/pkg/net/textproto/#CanonicalMIMEHeaderKey) names
must be specified.

The incoming request URL (received by the EventListener) is provided in the
`Eventlistener-Request-URL` header provided to the Webhook interceptor.

When multiple Interceptors are specified, requests are piped through each
Interceptor sequentially for processing - e.g. the headers/body of the first
Interceptor's response will be sent as the request to the second Interceptor. It
is the responsibility of Interceptors to preserve header/body data if desired.
The response body and headers of the last Interceptor is used for resource
binding/templating.

#### Event Interceptor Services

To be an Event Interceptor, a Kubernetes object should:

- Be fronted by a regular Kubernetes v1 Service over port 80
- Accept JSON payloads over HTTP
- Accept HTTP POST requests with JSON payloads.
- Return a HTTP 200 OK Status if the EventListener should continue processing
  the event
- Return a JSON body back. This will be used by the EventListener as the event
  payload for any further processing. If the Interceptor does not need to modify
  the body, it can simply return the body that it received.
- Return any Headers that might be required by other chained Interceptors or any
  bindings.

**Note**: It is the responsibility of Interceptors to preserve header/body data
if desired. The response body and headers of the last Interceptor is used for
resource binding/templating.

<!-- FILE: examples/eventlisteners/eventlistener-interceptor.yaml -->
```YAML
---
apiVersion: triggers.tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: listener-interceptor
spec:
  serviceAccountName: tekton-triggers-example-sa
  triggers:
    - name: foo-trig
      interceptors:
        - webhook:
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
        - ref: pipeline-binding
      template:
        ref: pipeline-template
```


### GitHub Interceptors

GitHub Interceptors contain logic to validate and filter webhooks that come from
GitHub. Supported features include validating webhooks actually came from GitHub
using the logic outlined in GitHub
[documentation](https://developer.github.com/webhooks/securing/), as well as
filtering incoming events.

To use this Interceptor as a validator, create a secret string using the method
of your choice, and configure the GitHub webhook to use that secret value.
Create a Kubernetes secret containing this value, and pass that as a reference
to the `github` Interceptor.

To use this Interceptor as a filter, add the event types you would like to
accept to the `eventTypes` field. Valid values can be found in GitHub
[docs](https://developer.github.com/webhooks/#events).

The body/header of the incoming request will be preserved in this Interceptor's
response.

```yaml
  triggers:
    - name: github-listener
      interceptors:
        - github:
            secretRef:
              secretName: github-secret
              secretKey: secretToken
            eventTypes:
 
```


Check out a full example of using GitHub Interceptor in [examples/github](../examples/github)

### GitLab Interceptors

GitLab Interceptors contain logic to validate and filter requests that come from
GitLab. Supported features include validating that a webhook actually came from
GitLab, using the logic outlined in GitLab
[documentation](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html),
and to filter incoming events based on the event types. Event types can be found
in GitLab
[documentation](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html#events).

To use this Interceptor as a validator, create a secret string using the method
of your choice, and configure the GitLab webhook to use that secret value.
Create a Kubernetes secret containing this value, and pass that as a reference
to the `gitlab` Interceptor.

To use this Interceptor as a filter, add the event types you would like to
accept to the `eventTypes` field.

The body/header of the incoming request will be preserved in this Interceptor's
response.

```yaml
apiVersion: triggers.tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: gitlab-listener-interceptor
spec:
  serviceAccountName: tekton-triggers-example-sa
  triggers:
    - name: foo-trig
      interceptors:
        - gitlab:
            secretRef:
              secretName: foo
              secretKey: bar
            eventTypes:
              - Push Hook
      bindings:
        - ref: pipeline-binding
      template:
        ref: pipeline-template
```

### Bitbucket Interceptors

The Bitbucket interceptor provides support for hooks originating in [Bitbucket server](https://confluence.atlassian.com/bitbucketserver), providing server hook signature validation and event-filtering.
[Bitbucket cloud](https://support.atlassian.com/bitbucket-cloud/) is not currently supported by this interceptor, as it has no secret validation, so you could match on the incoming requests using the CEL interceptor.

To use this Interceptor as a validator, create a secret string using the method
of your choice, and configure the Bitbucket webhook to use that secret value.
Create a Kubernetes secret containing this value, and pass that as a reference
to the `bitbucket` Interceptor.

To use this Interceptor as a filter, add the event types you would like to
accept to the `eventTypes` field. Valid values can be found in Bitbucket
[docs](https://confluence.atlassian.com/bitbucketserver/event-payload-938025882.html).

The body/header of the incoming request will be preserved in this Interceptor's
response.

<!-- FILE: examples/bitbucket/bitbucket-eventlistener-interceptor.yaml -->
```YAML
---
apiVersion: triggers.tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: bitbucket-listener
spec:
  serviceAccountName: tekton-triggers-bitbucket-sa
  triggers:
    - name: bitbucket-triggers
      interceptors:
        - bitbucket:
            secretRef:
              secretName: bitbucket-secret
              secretKey: secretToken
            eventTypes:
              - repo:refs_changed
      bindings:
        - ref: bitbucket-binding
      template:
        ref: bitbucket-template
```

### CEL Interceptors

CEL Interceptors can be used to filter or add extra information to incoming events, using the
[CEL](https://github.com/google/cel-go) expression language.

Please read the
[cel-spec language definition](https://github.com/google/cel-spec/blob/master/doc/langdef.md)
for more details on the expression language syntax.

The `cel-trig-with-matches` trigger below filters events that don't have an
`'X-GitHub-Event'` header matching `'pull_request'`.

It also modifies the incoming event, adding an extra key to the JSON body,
with a truncated string coming from the hook body.

```yaml
  triggers:
    - name: cel-trig-with-matches
      interceptors:
        - cel:
            filter: "header.match('X-GitHub-Event', 'pull_request')"
            overlays:
            - key: truncated_sha
              expression: "body.pull_request.head.sha.truncate(7)"
      bindings:
      - name: sha
        value: $(extensions.truncated_sha)
```

In addition to the standard expressions provided by CEL, Triggers supports some
useful functions for dealing with event data
[CEL expressions](./cel_expressions.md).


The `filter` expression must return a `true` value if this trigger is to be
processed, and the `overlays` applied.

Optionally, no `filter` expression can be provided, and the `overlays` will be
applied to the incoming body.
<!-- FILE: examples/eventlisteners/cel-eventlistener-no-filter.yaml -->
```YAML
apiVersion: triggers.tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: cel-eventlistener-no-filter
spec:
  serviceAccountName: tekton-triggers-example-sa
  triggers:
    - name: cel-trig
      interceptors:
        - cel:
            overlays:
            - key: extensions.truncated_sha
              expression: "body.pull_request.head.sha.truncate(7)"
      bindings:
      - ref: pipeline-binding
      template:
        ref: pipeline-template
```
#### Overlays

The CEL interceptor supports "overlays", these are CEL expressions whose values are added to the incoming event under a 
top-level `extensions` field  and are accessible by TriggerBindings.

```yaml
  triggers:
    - name: cel-trig
      interceptors:
        - cel:
            overlays:
            - key: extensions.truncated_sha
              expression: "body.pull_request.head.sha.truncate(7)"
            - key: extensions.branch_name
              expression: "body.ref.split('/')[2]"
```


In this example, the bindings will see two added" fields -- `extensions.truncated_sha` and `extensions.branch_name` in 
addition to the regular `body` and `header` fields.

The `key` element of the overlay can create new elements or replace existing elements within the extensions field.
Note that the original incoming body is not modified. Instead, new fields are added to a separate top-level `extensions`
field that is accessible by TriggerBindings. 

For example, this expression:

```YAML
- key: short_sha
  expression: "truncate(body.pull_request.head.sha, 7)"
```

Would see the `short_sha` being created in the `extensions` field:

```json
{
  "body": {
    "ref": "refs/heads/master",
    "pull_request": {
      "head": {
        "sha": "6113728f27ae82c7b1a177c8d03f9e96e0adf246"
      }
    }
  },
  "extensions": {
    "short_sha": "6113728"
  }
}
```

It's even possible to replace existing fields, by providing a key that matches
the path to an existing value.

Anything that is applied as an overlay can be extracted using a binding e.g.

<!-- FILE: examples/triggerbindings/cel-example-trigger-binding.yaml -->
```YAML
apiVersion: triggers.tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: pipeline-binding-with-cel-extensions
spec:
  params:
  - name: gitrevision
    value: $(extensions.branch_name)
  - name: branch
    value: $(extensions.short_sha)
```

## EventListener Response

The EventListener responds with 201 Created status code when at least one of the trigger is executed successfully. Otherwise, it returns 202 Accepted status code.
The EventListener responds with following message after receiving the event:
```JSON
{"eventListener":"listener","namespace":"default","eventID":"h2bb7"}
```
- `eventListener` - Refers to the EventListener Name.
- `namespace` - Refers to the namespace of the EventListener
- `eventID` - Refers to the uniqueID that gets assigned to each incoming request

## How does the EventListener work?

Lets understand how an EventListener works with an example using GitHub

* Create a sample GitHub example
```bash
kubectl create -f https://github.com/tektoncd/triggers/tree/master/examples/github
```

* Once the EventListener is created,  the Triggers controller will create a new `Deployment` and `Service` for the EventListener. We can use `kubectl` to see them running:
```bash
kubectl get deployment
NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
el-github-listener-interceptor   1/1     1            1           11s

kubectl get svc
NAME                             TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
el-github-listener-interceptor   ClusterIP   10.99.188.140   <none>        8080/TCP   52s
```
The Triggers controller uses fields from the EventListener's `spec` (which is described in the [Syntax](https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#syntax) section, as well as [`metadata.labels`](https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#labels)
in addition to some pre-configured information like (container `Image`, `Name`, `Port`) to create the **Deployment** and **Service**.

We follow a naming convention while creating these resources. An EventListener named `foo` will create a deployment and a service both named `el-foo`.

Once all the resources are up and running user can get a URL to send webhook events. This URL points to the service created above and points to the deployment.
```bash
kubectl get eventlistener
NAME                          ADDRESS                                                              AVAILABLE   REASON
github-listener-interceptor   http://el-github-listener-interceptor.ptest.svc.cluster.local:8080   True        MinimumReplicasAvailable
```

Follow [GitHub example](https://github.com/tektoncd/triggers/blob/master/examples/github/README.md) to try out locally.

## Examples

For complete examples, see
[the examples folder](https://github.com/tektoncd/triggers/tree/master/examples).

## Response Timeout
Eventlistener sink timeout if EventListener isn't able to process the request within certain duration. The response timeout configuration is defined in [controller.yaml](../config/controller.yaml).
- `-el-readtimeout`: This define ReadTimeOut for sink server. Default value is 5 s.
- `-el-writetimeout`: This define WriteTimeout for sink server. Default value is 40s.
- `-el-idletimeout`: This define the IdleTimeout for sink server. Default value is 120s.
- `-el-timeouthandler`: This define the Timeout for Handler for sink server's route. Default value is 30s.


## Multi-Tenant Concerns

The EventListener is effectively an additional form of client into Tekton, versus what 
example usage via `kubectl` or `tkn` which you have seen elsewhere.  In particular, the HTTP based
events bypass the normal Kubernetes authentication path you get via `kubeconfig` files 
and the `kubectl config` family of commands.

As such, there are set of items to consider when deciding how to 

- best expose (each) EventListener in your cluster to the outside world.
- best control how (each) EventListener and the underlying API Objects described below access, create,
and update Tekton related API Objects in your cluster.

Minimally, each EventListener has its [ServiceAccountName](#serviceAccountName) as noted below and all
events coming over the "Sink" result in any Tekton resource interactions being done with the permissions 
assigned to that ServiceAccount.

However, if you need differing levels of permissions over a set of Tekton resources across the various
[Triggers](#triggers) and [Interceptors](#Interceptors), where not all Triggers or Interceptors can 
manipulate certain Tekton Resources in the same way, a simple, single EventListener will not suffice.

Your options at that point are as follows:

### Multiple EventListeners (One EventListener Per Namespace)

You can create multiple EventListener objects, where your set of Triggers and Interceptors are spread out across the 
EventListeners.

If you create each of those EventListeners in their own namespace, it becomes easy to assign 
varying permissions to the ServiceAccount of each one to serve your needs.  And often times namespace
creation is coupled with a default set of ServiceAccounts and Secrets that are also defined.
So conceivably some administration steps are taken care of.  You just update the permissions
of the automatically created ServiceAccounts.

Possible drawbacks:
- Namespaces with associated Secrets and ServiceAccounts in an aggregate sense prove to be the most expensive
items in Kubernetes underlying `etcd` store.  In larger clusters `etcd` storage capacity can become a concern.
- Multiple EventListeners means multiple HTTP ports that must be exposed to the external entities accessing 
the "Sink".  If you happen to have a HTTP Firewall between your Cluster and external entities, that means more
administrative cost, opening ports in the firewall for each Service, unless you can employ Kubernetes `Ingress` to
serve as a routing abstraction layer for your set of EventListeners. 

### Multiple EventListeners (Multiple EventListeners per Namespace)

Multiple EventListeners per namespace will most likely mean more ServiceAccount/Secret/RBAC manipulation for
the administrator, as some of the built in generation of those artifacts as part of namespace creation are not
applicable.

However you will save some on the `etcd` storage costs by reducing the number of namespaces.

Multiple EventListeners and potential Firewall concerns still apply (again unless you employ `Ingress`).

### ServiceAccount per EventListenerTrigger

Being able to set a ServiceAccount on an EventListenerTrigger allows for finer grained permissions as well.

You still have to create the additional ServiceAccounts.

But staying within 1 namespace and minimizing the number of EventListeners with their associated "Sinks" minimizes 
concerns around `etcd` storage and port considerations with Firewalls if `Ingress` is not utilized.

---

Except as otherwise noted, the content of this page is licensed under the
[Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/),
and code samples are licensed under the
[Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0).

## EventListener Secure Connection

Triggers now support both `HTTP` and `HTTPS` connection by adding few configuration to eventlistener.

To setup TLS connection add two set of reserved environment variables `TLS_CERT` and `TLS_KEY` using `secretKeyRef` env type 
where we need to specify the `secret` which contains `cert` and `key` files. See the full [example]((../examples/eventlistener-tls-connection/README.md)) for more details.

Refer [TEP-0027](https://github.com/tektoncd/community/blob/master/teps/0027-https-connection-to-triggers-eventlistener.md) for more information on design and user stories.
