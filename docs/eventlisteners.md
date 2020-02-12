# EventListener

EventListener is a Kubernetes custom resource that allows users a declarative
way to process incoming HTTP based events with JSON payloads. EventListeners
expose an addressable "Sink" to which incoming events are directed. Users can
declare [TriggerBindings](./triggerbindings.md) to extract fields from events,
and apply them to [TriggerTemplates](./triggertemplates.md) in order to create
Tekton resources. In addition, EventListeners allow lightweight event processing
using [Event Interceptors](#Interceptors).

- [Syntax](#syntax)
  - [Triggers](#triggers)
    - [Interceptors](#Interceptors)
  - [ServiceAccountName](#serviceAccountName)
- [Logging](#logging)
- [Labels](#labels)
- [Examples](#examples)

## Syntax

To define a configuration file for an `EventListener` resource, you can specify
the following fields:

- Required:
  - [`apiVersion`][kubernetes-overview] - Specifies the API version, for example
    `tekton.dev/v1alpha1`.
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

[kubernetes-overview]:
  https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields

### Triggers

The `triggers` field is required. Each EventListener can consist of one or more
`triggers`. A Trigger consists of:

- `name` - (Optional) a valid
  [Kubernetes name](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set)
- [`interceptors`](#interceptors) - (Optional) list of interceptors to use
- `bindings` - A list of names of `TriggerBindings` to use
- `template` - The name of `TriggerTemplate` to use

```yaml
triggers:
  - name: trigger-1
    interceptors:
      - github:
          eventTypes: ["pull_request"]
    bindings:
      - name: pipeline-binding
      - name: message-binding
    template:
      name: pipeline-template
```

### ServiceAccountName

The `serviceAccountName` field is required. The ServiceAccount that the
EventListener sink uses to create the Tekton resources. The ServiceAccount needs
a role with the following rules:

<!-- FILE: examples/role-resources/triggerbinding-roles/role.yaml -->

```YAML
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: tekton-triggers-example-minimal
rules:
# Permissions for every EventListener deployment to function
- apiGroups: ["tekton.dev"]
  resources: ["eventlisteners", "triggerbindings", "triggertemplates"]
  verbs: ["get"]
- apiGroups: [""]
  resources: ["configmaps", "secrets"] # secrets are only needed for Github/Gitlab interceptors
  verbs: ["get", "list", "watch"]
# Permissions to create resources in associated TriggerTemplates
- apiGroups: ["tekton.dev"]
  resources: ["pipelineruns", "pipelineresources", "taskruns"]
  verbs: ["create"]
```

If your EventListener is using
[`ClusterTriggerBindings`](./clustertriggerbindings.md), you'll need a
ServiceAccount with a
[ClusterRole instead](../examples/role-resources/clustertriggerbinding-roles/clusterrole.yaml).

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

## Labels

By default, EventListeners will attach the following labels automatically to all
resources it creates:

| Name                     | Description                                            |
| ------------------------ | ------------------------------------------------------ |
| tekton.dev/eventlistener | Name of the EventListener that generated the resource. |
| tekton.dev/trigger       | Name of the Trigger that generated the resource.       |
| tekton.dev/eventid       | UID of the incoming event.                             |

Since the EventListener name and Trigger name are used as label values, they
must adhere to the
[Kubernetes syntax and character set requirements](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set)
for label values.

## Interceptors

Triggers within an `EventListener` can optionally specify interceptors, to
modify the behavior or payload of Triggers.

Event Interceptors can take several different forms today:

- [Webhook Interceptors](#Webhook-Interceptors)
- [GitHub Interceptors](#GitHub-Interceptors)
- [GitLab Interceptors](#GitLab-Interceptors)
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
apiVersion: tekton.dev/v1alpha1
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
        - name: pipeline-binding
      template:
        name: pipeline-template
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

<!-- FILE: examples/eventlisteners/github-eventlistener-interceptor.yaml -->

```YAML
---
apiVersion: tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: github-listener-interceptor
spec:
  serviceAccountName: tekton-triggers-example-sa
  triggers:
    - name: foo-trig
      interceptors:
        - github:
            secretRef:
              secretName: foo
              secretKey: bar
            eventTypes:
              - pull_request
      bindings:
        - name: pipeline-binding
      template:
        name: pipeline-template
```

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
apiVersion: tekton.dev/v1alpha1
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
        - name: pipeline-binding
      template:
        name: pipeline-template
```

### CEL Interceptors

CEL Interceptors parse expressions to filter requests based on JSON bodies and
request headers, using the [CEL](https://github.com/google/cel-go) expression
language. Please read the
[cel-spec language definition](https://github.com/google/cel-spec/blob/master/doc/langdef.md)
for more details on the expression language syntax.

In addition to the standard
[CEL expression language syntax](https://github.com/google/cel-spec/blob/master/doc/langdef.md),
Triggers supports these additional [CEL expressions](./cel_expressions.md).

The body/header of the incoming request will be preserved in this Interceptor's
response.

<!-- FILE: examples/eventlisteners/cel-eventlistener-interceptor.yaml -->

```YAML
apiVersion: tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: cel-listener-interceptor
spec:
  serviceAccountName: tekton-triggers-example-sa
  triggers:
    - name: cel-trig-with-matches
      interceptors:
        - cel:
            filter: "header.match('X-GitHub-Event', 'pull_request')"
            overlays:
            - key: extensions.truncated_sha
              expression: "truncate(body.pull_request.head.sha, 7)"
      bindings:
      - name: pipeline-binding
      template:
        name: pipeline-template
    - name: cel-trig-with-canonical
      interceptors:
        - cel:
            filter: "header.canonical('X-GitHub-Event') == 'push'"
      bindings:
      - name: pipeline-binding
      template:
        name: pipeline-template
```

If no filter is provided, then the overlays will be applied to the body. With a
filter, the `expression` must return a `true` value, otherwise the request will
be filtered out.

<!-- FILE: examples/eventlisteners/cel-eventlistener-no-filter.yaml -->

```YAML
apiVersion: tekton.dev/v1alpha1
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
              expression: "truncate(body.pull_request.head.sha, 7)"
      bindings:
      - name: pipeline-binding
      template:
        name: pipeline-template
```

## Examples

For complete examples, see
[the examples folder](https://github.com/tektoncd/triggers/tree/master/examples).

---

Except as otherwise noted, the content of this page is licensed under the
[Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/),
and code samples are licensed under the
[Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0).
