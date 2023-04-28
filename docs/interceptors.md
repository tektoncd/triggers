<!--
---
linkTitle: "Configuring Interceptors"
weight: 5
---
-->
# Configuring `Interceptors`

- [Overview](#overview)
- [Specifying an `Interceptor`](#specifying-an-interceptor)
- [Webhook `Interceptors`](#webhook-interceptors)
- [GitHub `Interceptors`](#github-interceptors)
- [GitLab `Interceptors`](#gitlab-interceptors)
- [Bitbucket `Interceptors`](#bitbucket-interceptors)
  - [Bitbucket Server](#bitbucket-server)
  - [Bitbucket Cloud](#bitbucket-cloud)
- [slack `Interceptors`](#slack-interceptors)
- [CEL `Interceptors`](#cel-interceptors)
- [Implementing custom `Interceptors`](#implementing-custom-interceptors)

## Overview

An `Interceptor` is a "catch-all" event processor for a specific platform that runs before the `TriggerBinding.` It allows you to perform payload filtering, verification (using a secret),
transformation, define and test trigger conditions, and implement other useful processing. Once the event data passes through an `Interceptor`, it then goes to the `Trigger` before you pass
the payload data to the `TriggerBinding`. You can also use an `Interceptor` to modify the behavior of the associated `Trigger`.

Tekton Triggers currently supports two distinct `Interceptor` implementations:
- Standalone `Interceptors`, which are instances of the [`Interceptor`](./namespacedinterceptor) or the [`ClusterInterceptor`](./clusterinterceptors.md) Custom Resource Definition (CRD). You specify these `Interceptors` by referencing them,
  along with the desired parameters, within your  `EventListener`. You can use the `ClusterInterceptor` CRD to implement your own custom `Interceptors`.
- Legacy `Interceptors`, which you define entirely as part of the `EventListener` definition. This implementation will eventually be deprecated, so please consider
  transitioning to standalone `Interceptors` as soon as possible. See [TEP-0026](https://github.com/tektoncd/community/blob/main/teps/0026-interceptor-plugins.md) for more context on this change.

Tekton Triggers ships with the following `Interceptors` to help you get started:
- [Webhook `Interceptors`](#webhook-interceptors)
- [GitHub `Interceptors`](#github-interceptors)
- [GitLab `Interceptors`](#gitlab-interceptors)
- [Bitbucket `Interceptors`](#bitbucket-interceptors)
  - [Bitbucket Server](#bitbucket-server)
  - [Bitbucket Cloud](#bitbucket-cloud)
- [CEL `Interceptors`](#cel-interceptors)

## Specifying an `Interceptor`

To specify an `Interceptor` within your `EventListener`, create an `interceptors:` field with the following sub-fields:
- `name` - (optional) a name that uniquely identifies this `Interceptor` definition
- `ref` - a reference to a [`ClusterInterceptor`](./clusterinterceptors.md) or [`Interceptor`](./namespacedinterceptors.md) object with the following fields:
  - `name` - the name of the referenced `ClusterInterceptor`
  - `kind` - (optional) specifies that whether the referenced Kubernetes object is a `ClusterInterceptor` object or `NamespacedInterceptor`. Default value is `ClusterInterceptor`
  - `apiVersion` - (optional) specifies the target API version, for example `triggers.tekton.dev/v1alpha1`
  - `params` - `name`/`value` pairs that specify the parameters you want to pass to the `ClusterInterceptor`
- `params` - (optional) `name`/`value` pairs that specify the desired parameters for the `Interceptor`;
  the `name` field takes a string, while the `value` field takes a valid JSON object

Below is an example standalone `Interceptor` reference within an `EventListener` definition:

```yaml
interceptors:
    - name: "validate GitHub payload and filter on eventType"
      ref:
        name: "github"
      params:
      - name: "secretRef"
        value:
          secretName: github-secret
          secretKey: secretToken
      - name: "eventTypes"
        value: ["pull_request"]
    - name: "CEL filter: only when PRs are opened"
      ref:
        name: "cel"
      params:
      - name: "filter"
        value: "body.action in ['opened', 'reopened']"
```

### Webhook `Interceptors`

**Note:** Tekton Triggers ships with only a legacy Webhook `Interceptor`. If you want to implement it using 
the standalone model, see [Implementing custom `Interceptors`](#implementing-custom-interceptors).

A Webhook `Interceptor` allows you to process your event payload by an external Kubernetes object containing
custom business logic. The Kubernetes object, exposed via a Kubernetes Service, receives event payload
data from your `EventListener` via HTTP, applies its business logic to it, and returns the processed payload
(both headers plus body) via a HTTP 200 response to the `EventListener`.

This payload can then continue on to the `TriggerBinding` specified in the `EventListener`. If processing
is not successful, the `Interceptor` recognizes that the returned HTTP response code is not 200 and halts
further processing of the event payload. 

You can optionally specify additional data to merge with the event payload before sending out for processing
by adding the data as [canonical](https://golang.org/pkg/net/textproto/#CanonicalMIMEHeaderKey) key/value pairs
in the `Interceptor's` `header` field.

Your external Kubernetes object must meet the following criteria:

- Fronted by a regular Kubernetes v1 Service running on HTTP port 80,
- Accepts JSON payloads over HTTP,
- Accepts HTTP POST requests with JSON payloads,
- Returns an HTTP 200 OK response when you want the `EventListener` to continue processing the event,
- Returns a JSON object in the HTTP response body. 
- Returns headers expected by subsequently chained `Interceptors` and the associated `TriggerBinding`.

**Note:** If your business logic does not modify either the HTTP payload's header or body, simply return the same HTTP 
header or body that you received.

Below is an example Webhook `Interceptor` definition:

```yaml
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

A GitHub `Interceptor` contains logic that validates and filters GitHub webhooks.
It can validate the webhook's origin as described in [Securing your webhooks](https://developer.github.com/webhooks/securing/)
as well as filter incoming events by the criteria you specify. The GitHub `Interceptor`
always preserves the payload data (both header and body) in its responses.

To use a GitHub `Interceptor` as a GitHub webhook validator, do the following:

1. Create a secret string value.
2. Configure the `GitHub` webhook with that value.
3. Create a Kubernetes secret containing your secret value.
4. Pass the Kubernetes secret as a reference to your GitHub `Interceptor`.

To use a GitHub `Interceptor` as a filter for event data, specify the event types
you want the `Interceptor` to accept in the `eventTypes` field. The `Interceptor`
accepts data event types listed in [Event types and payloads](https://docs.github.com/en/developers/webhooks-and-events/webhook-events-and-payloads).

Below is an example GitHub `Interceptor` reference:

```yaml
 triggers:
    - name: github-listener
      interceptors:
        - ref:
            name: "github"
            kind: ClusterInterceptor
            apiVersion: triggers.tekton.dev
          params:
          - name: "secretRef"
            value:
              secretName: github-secret
              secretKey: secretToken
          - name: "eventTypes"
            value: ["pull_request"]
```

For reference, below is an example legacy GitHub `Interceptor` definition:

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

For more information, see our [example](../examples/v1beta1/github) of using this `Interceptor`.

#### Adding Changed Files

The GitHub `Interceptor` also has the ability to add a comma delimited list of all files that have changed (added, modified or deleted) for the `push` and `pull_request` events. The list of changed files are added to the `changed_files` property of the event payload in the top-level `extensions` field

Below is an example GitHub `Interceptor` that enables the `addChangedFiles` feature and uses the CEL `Interceptor` to filter incoming events by the files changed

```yaml
 triggers:
    - name: github-listener
      interceptors:
        - ref:
            name: "github"
            kind: ClusterInterceptor
            apiVersion: triggers.tekton.dev
          params:
          - name: "secretRef"
            value:
              secretName: github-secret
              secretKey: secretToken
          - name: "eventTypes"
            value: ["pull_request", "push"]
          - name: "addChangedFiles"
            value:
              enabled: true
        - ref:
            name: cel
          params:
          - name: filter
            # execute only when a file within the controllers directory has changed
            value: extensions.changed_files.matches('controllers/')
```

##### Adding Changed Files - Private Repository

The ability to add changed files can also work with private repositories by supplying a [GitHub personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token) in the `personalAccessToken` field. In the example below, the `personalAccessToken` uses the `github-pat` secret to get the GitHub personal access token used to make the API calls to get the list of changed files.

```yaml
 triggers:
    - name: github-listener
      interceptors:
        - ref:
            name: "github"
            kind: ClusterInterceptor
            apiVersion: triggers.tekton.dev
          params:
          - name: "secretRef"
            value:
              secretName: github-secret
              secretKey: secretToken
          - name: "eventTypes"
            value: ["pull_request", "push"]
          - name: "addChangedFiles"
            value:
              enabled: true
              personalAccessToken:
                secretName: github-pat
                secretKey: token
        - ref:
            name: cel
          params:
          - name: filter
            # execute only when a file within the controllers directory has changed
            value: extensions.changed_files.matches('controllers/')
```

For more information around adding changed files, see the following examples

- [github-add-changed-files-pr](../examples/v1beta1/github-add-changed-files-pr)
- [github-add-changed-files-push-cel](../examples/v1beta1/github-add-changed-files-push-cel)

#### Owners validation for pull requests

The GitHub `Interceptor` supports the ability to halt processing on a pull request if the user is not listed in the [owners](https://www.kubernetes.dev/docs/guide/owners/) file or is not a repository/organization member/owner. This feature can be used to prevent unnecessary execution of a PipelineRun or TaskRun. The GitHub `Interceptor` also supports the ability to trigger a PipelineRun/TaskRun through a comment that contains `/ok-to-test` on a pull request by an owner. 

This feature will also work against private GitHub repositories by supplying a [GitHub personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token) in the `personalAccessToken` field.

> NOTE: Owners validation requires (at a minimum) the `pull_request` and `issue_comment` GitHub event types.

To use the GitHub `Interceptor` as a GitHub owners validator, do the following:

1. Create a secret string value.
2. Configure the `GitHub` webhook with that value.
3. Create a Kubernetes secret containing your secret value called `SecretRef`.
4. Pass the Kubernetes secret as a reference to your GitHub `Interceptor`.
5. Create an owners file and add the list of approvers into the approvers section.

Below is an example GitHub `Interceptor` using owners validation:

```yaml
 triggers:
    - name: github-listener
      interceptors:
        - ref:
            name: "github"
            kind: ClusterInterceptor
            apiVersion: triggers.tekton.dev
          params:
            - name: "secretRef"
              value:
                secretName: github-secret
                secretKey: secretToken
            - name: "eventTypes"
              # Owners validation requires (at a minimum) the `pull_request` and `issue_comment` GitHub event types
              value: ["pull_request", "issue_comment"]
            - name: "githubOwners"
              value: 
                enabled: true
                # This value is needed for private repos or when checkType is set to orgMembers or repoMembers or all
                # personalAccessToken:
                #   secretName: github-token
                #   secretKey: secretToken
                checkType: none
```

For more information around owners file validation, see the following example

- [github-owners](../examples/v1beta1/github-owners)

### GitLab Interceptors

A GitLab `Interceptor` contains logic that validates and filters GitLab webhooks.
It can validate the webhook's origin as described in [Webhooks](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html)
as well as filter incoming events by the criteria you specify. The GitLab `Interceptor`
always preserves the payload data (both header and body) in its responses.

To use a GitLab `Interceptor` as a GitLab webhook validator, do the following:

1. Create a secret string value.
2. Configure the GitLab webhook with that value.
3. Create a Kubernetes secret containing your secret value.
4. Pass the Kubernetes secret as a reference to your GitLab `Interceptor`.

To use a GitLab `Interceptor` as a filter for event data, specify the event types
you want the `Interceptor` to accept in the `eventTypes` field. The `Interceptor`
accepts data event types listed in [Events](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html#events).

Below is an example GitLab `Interceptor` reference:

```yaml
interceptors:
- ref:
    name: "gitlab"
  params:
  - name: "secretRef"
    value:
      secretName: foo
      secretKey: bar
  - name: "eventTypes"
    value: ["Push Hook"]
```

For reference, below is an example legacy GitLab `Interceptor` definition:

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

### Bitbucket `Interceptors`

Bitbucket `Interceptors` has support for both Bitbucket server (which does secret validation and event filtering) and Bitbucket cloud (which does event filtering).

### Bitbucket Server

A Bitbucket server contains logic that validates and filters [Bitbucket server](https://confluence.atlassian.com/bitbucketserver) webhooks.
It can validate the webhook's origin as described in [Webhooks](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html). The Bitbucket `Interceptor` always preserves the payload
data (both header and body) in its responses.

To use a Bitbucket `Interceptor` as a Bitbucket server webhook validator, do the following:

1. Create a secret string value.
2. Configure the Bitbucket server webhook with that value.
3. Create a Kubernetes secret containing your secret value.
4. Pass the Kubernetes secret as a reference to your Bitbucket `Interceptor`.

To use a Bitbucket `Interceptor` as a filter for event data, specify the event types
you want the `Interceptor` to accept in the `eventTypes` field. The `Interceptor`
accepts data event types listed in [Event payload](https://confluence.atlassian.com/bitbucketserver/event-payload-938025882.html).

Below is an example Bitbucket `Interceptor` reference for server:

```yaml
interceptors:
- ref:
    name: "bitbucket"
  params:
    - name: secretRef
      value:
        secretName: bitbucket-server-secret
        secretKey: secretToken
    - name: eventTypes
      value:
        - repo:refs_changed
```

For reference, below is an example legacy Bitbucket `Interceptor` definition:

```yaml
---
apiVersion: triggers.tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: bitbucket-server-listener
spec:
  serviceAccountName: tekton-triggers-example-sa
  triggers:
    - name: bitbucket-server-triggers
      interceptors:
        - ref:
            name: "bitbucket"
          params:
            - name: secretRef
              value:
                secretName: bitbucket-server-secret
                secretKey: secretToken
            - name: eventTypes
              value:
                - repo:refs_changed
      bindings:
        - ref: bitbucket-server-binding
      template:
        ref: bitbucket-server-template
```

### Bitbucket Cloud

At the moment, Bitbucket cloud does not support validating webhook payloads using shared secrets. 
See the [Secure Webhooks section](https://support.atlassian.com/bitbucket-cloud/docs/manage-webhooks/) of Bitbucket cloud docs for more information on securing your Bitbucket webhooks.

To use a Bitbucket `Interceptor` as a filter for event data, specify the event types
you want the `Interceptor` to accept in the `eventTypes` field. The `Interceptor`
accepts data event types listed in [Event payload](https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/).

Below is an example Bitbucket `Interceptor` reference for cloud:

```yaml
interceptors:
- ref:
    name: "bitbucket"
  params:
    - name: eventTypes
      value:
        - repo:push
```

For reference, below is an example legacy Bitbucket `Interceptor` definition:

```yaml
---
apiVersion: triggers.tekton.dev/v1beta1
kind: EventListener
metadata:
  name: bitbucket-cloud-listener
spec:
  serviceAccountName: tekton-triggers-example-sa
  triggers:
    - name: bitbucket-cloud-triggers
      interceptors:
        - ref:
            name: "bitbucket"
          params:
            - name: eventTypes
              value:
                - repo:push
      bindings:
        - ref: bitbucket-cloud-binding
      template:
        ref: bitbucket-cloud-template
```

### Slack Interceptors
A Slack `Interceptor` allows you to extract fields from a slack slash command [payload](https://api.slack.com/interactivity/slash-commands#app_command_handling) which are sent in the  http form-data section. 
the `Interceptor` requests fields extracts the requested fields and appends them to the `extensions`. 


```yaml
apiVersion: triggers.tekton.dev/v1beta1
kind: EventListener
metadata:
  name: slack-listener
  annotations:
    tekton.dev/payload-validation: "false"
spec:
  triggers:
    - name: slack-trigger
      interceptors:
        - ref:
            name: "slack"
            kind: ClusterInterceptor
          params:
            - name: requestedFields
              value: 
                - text   
```
Note: payload-validation must be disabled by add adding the following annotation 

```yaml
  annotations:
    tekton.dev/payload-validation: "false"
```


### CEL Interceptors

A CEL `Interceptor` allows you to filter and modify the payloads of incoming events using
the [CEL](https://github.com/google/cel-spec/blob/master/doc/langdef.md) expression language.

CEL `Interceptors` support `overlays`, which are CEL expressions that Tekton Triggers adds
to the event payload in the top-level `extensions` field. `overlays` are accessible from
`TriggerBindings`.

In the example `overlays` definition below, the `Interceptor` adds two new fields to the
event payload that the corresponding `TriggerBinding` will receive in addition to the standard
`header` and `body fields`: `extensions.truncated_sha` and `extensions.branch_name`:

```yaml
  triggers:
    - name: cel-trig
      interceptors:
        - ref:
            name: cel
          params:
          - name: "overlays"
            value:
            - key: truncated_sha
              expression: "body.pull_request.head.sha.truncate(7)"
            - key: branch_name
              expression: "body.ref.split('/')[2]"
```

Below is the same example as a legacy `Interceptor`:

```yaml
  triggers:
    - name: cel-trig
      interceptors:
        - cel:
            overlays:
            - key: truncated_sha
              expression: "body.pull_request.head.sha.truncate(7)"
            - key: branch_name
              expression: "body.ref.split('/')[2]"
```

You can use the `key` element within the `overlays` definition to create new or replace existing
elements within the `extensions` field. This does not modify the body of the event payload, but instead
adds the extra fields to the top-level `extensions` field.

For example, the following expression:

```yaml
- key: short_sha
  expression: "truncate(body.pull_request.head.sha, 7)"
```

can access the `short_sha` field and its value that have been created in `extensions` field:

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

**Note:** You can also replace existing fields by specifying a key that matches the path to an existing field/value pair.

You can access the extra fields added by a CEL `Interceptor` from your `TriggerBinding` as follows:

```yaml
apiVersion: triggers.tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: pipeline-binding-with-cel-extensions
spec:
  params:
  - name: gitrevision
    value: $(extensions.short_sha)
  - name: branch
    value: $(extensions.branch_name)
```

In the example CEL `Interceptor` definition below, the `cel-trig-with-matches` `Trigger`
filters events that don't have an `'X-GitHub-Event'` header matching `'pull_request'` and
adds an extra key to the JSON body of the payload with a truncated string derived from the hook
body:

```yaml
  triggers:
    - name: cel-trig-with-matches
      interceptors:
        - ref:
            name: "cel"
          params:
          - name: "filter"
            value: "header.match('X-GitHub-Event', 'pull_request')"
          - name: "overlays"
            value:
              - key: truncated_sha
                expression: "body.pull_request.head.sha.truncate(7)"
      bindings:
      - name: sha
        value: $(extensions.truncated_sha)
```

Below is the same example as a legacy `Interceptor`:

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

In the example CEL `Interceptor` definition below, the `filter` expression must
return a `true` value for this `Trigger` to execute and apply the specified `overlays`:

You also have the option to omit the `filter` expression entirely, in which case the `Interceptor`
applies the specified `overlays` to the payload's body:

```yaml
apiVersion: triggers.tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: cel-eventlistener-no-filter
spec:
  serviceAccountName: tekton-triggers-example-sa
  triggers:
    - name: cel-trig
      interceptors:
        - ref:
            name: "cel"
          params:
            - name: "overlays"
              value:
                - key: extensions.truncated_sha
                  expression: "body.pull_request.head.sha.truncate(7)"
      bindings:
      - ref: pipeline-binding
      template:
        ref: pipeline-template
```

### Chaining `Interceptors`

You can chain `Interceptors` with the following constraints:

- `ClusterInterceptors` do not modify the body of the event payload; instead, they add extra fields to the top-level `extensions` field.

- Webhook `Interceptors` can modify the body of the event payload, but cannot access the top-level `extensions` field.

#### Chaining ClusterInterceptors

Each ClusterInterceptor can return values in the InterceptorResponse within the `extensions` field. These values are then added to the `extensions` field of the InterceptorRequest that is sent to the next interceptor in the chain. 

If two interceptors return an extensions field with the same name, the latter one will overwrite the one from the previous one i.e. if interceptors A and B both return `foo` in the Extensions field of the InterceptorResponse, the values written by B will overwrite the ones written by A. To prevent this, it is recommended that each cluster interceptor write to its own top level field i.e A returns `A.foo` and B return `B.foo` in the InterceptorResponse.

#### Chaining Webhook Interceptors

**Note:** We are working on changing the behavior of Webhook `Interceptors` to match that of CEL `Interceptors` so that both `Interceptor` types can share data via the top-level `extensions` field.

Since Webhook `Interceptors` cannot access the `extensions` field, the `EventListener` adds the `extensions` field to the body of the event payload before passing it to any Webhook `Interceptor`.
In the example `Interceptor` chain shown below, the first CEL `Interceptor` adds the `truncated_sha` field to the `extensions` field and the `EventListener` adds it to the body of the payload so that
the Webhook `Interceptor` can access that data:
  
  ```yaml
    interceptors:
      - cel:
          overlays:
            - key: "truncated_sha"
              expression: "body.sha.truncate(5)"
      - webhook:
        objectRef:
          kind: Service
          name: some-interceptor
          apiVersion: v1
      - cel:
          filter: "body.extensions.truncated_sha == \"abcde\"" # Can also be extensions.truncated_sha == \"abcde\"
  ```

As a result, the body of the payload sent to the Webhook `Interceptor` is as follows:

```
{
  "sha": "abcdefghi", // Original field
  "extensions": {
     "truncated_sha": "abcde" 
  }
}
```

As long as the Webhook `Interceptor` does not modify the body of the payload, the last CEL interceptor in the chain and the target `TriggerBinding` can access the `truncated_sha`
field both in the body of the payload as well as via the extra fields added to the top-level `extension` field, namely `$(body.extensions.truncated_sha)` as well as `$(extensions.truncated_sha)`.

## Implementing custom `Interceptors`

Tekton Triggers ships with the `ClusterInterceptor`and `Interceptor` Custom Resource Definition (CRD), which you can use to implement custom `Interceptors`. See [`ClusterInterceptors`](./clusterinterceptors.md) and [`NamespacedInterceptors`](./namespacedinterceptors.md)  for more information.
