<!--
---
linkTitle: "Trigger Bindings"
weight: 4
---
-->
# `TriggerBindings`

A `TriggerBinding` is a resource that specifies the fields in the event payload from which you want to extract data as well as the fields in your
corresponding [`TriggerTemplate`](.docs/triggertemplates.md) to populate with the extracted values. In other words, it *binds* payload fields to
fields in the [`TriggerTemplate`] and for this to work the fields specified in the [`TriggerBinding`] And the corresponding [`TriggerTemplate`]
**must** match. You can then use the populated fields in your [`TriggerTemplate`] to populate fields in the [`TaskRun`] or [`PipelineRun`] associated
with that [`TriggerTemplate`]. Tekton also supports a cluster-scoped version called a `ClusterTriggerBinding` to encourage reusability across your
entire cluster.

## Structure of a `TriggerBinding`

Below is an example `TriggerBinding` definition:

<!-- FILE: examples/triggerbindings/triggerbinding.yaml -->
```YAML
apiVersion: triggers.tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: pipeline-binding
spec:
  params:
  - name: gitrevision
    value: $(body.head_commit.id)
  - name: gitrepositoryurl
    value: $(body.repository.url)
  - name: contenttype
    value: $(header.Content-Type)
```

## `TriggerBindings` vs. `ClusterTriggerBindings`

A `ClusterTriggerBinding` is a cluster-scoped `TriggerBinding` that you can reuse across your entire cluster.
You can reference a `ClusterTriggerBinding` in any `EventListener` in any namespace. You can specify multiple
`ClusterTriggerBindings` within your `Trigger` as well as specify the same `ClusterTriggerBinding` in multiple
`Triggers`.

Below is an example `ClusterTriggerBinding` definition:

<!-- FILE: examples/clustertriggerbindings/clustertriggerbinding.yaml -->
```YAML
apiVersion: triggers.tekton.dev/v1alpha1
kind: ClusterTriggerBinding
metadata:
  name: pipeline-clusterbinding
spec:
  params:
    - name: gitrevision
      value: $(body.head_commit.id)
    - name: gitrepositoryurl
      value: $(body.repository.url)
    - name: contenttype
      value: $(header.Content-Type)
```

When referencing a `ClusterTriggerBinding`, you must specify a `kind` value within the `bindings` field.
The default is `TriggerBinding` which denotes a namespaced `TriggerBinding`. For example:

<!-- FILE: examples/eventlisteners/eventlistener-clustertriggerbinding.yaml -->
```YAML
---
apiVersion: triggers.tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: listener-clustertriggerbinding
spec:
  serviceAccountName: tekton-triggers-example-sa
  triggers:
    - name: foo-trig
      bindings:
        - ref: pipeline-clusterbinding
          kind: ClusterTriggerBinding
        - ref: message-clusterbinding
          kind: ClusterTriggerBinding
      template:
        ref: pipeline-template
```

## Specifying paramters

A `TriggerBinding` allows you to specify parameters (`params`) that Tekton passes to the corresponding `TriggerTemplate`.
For each parameter, you must specify a `name` and a `value` field with the appropriate values.

## Accessing data in HTTP JSON payloads

Tekton can use a `TriggerBinding` to access data in the headers and body of an HTTP JSON payload. To do so, it uses
JSONPath expressions encapsulated within a `$()` wrapper. Keys in HTTP JSON headers are case-sensitive.

For example, below is a valid expression:

```shell
$(body.key1)
$(.body.key)
```

On the other hand, the expressions below are invalid:

```shell
.body.key1 # INVALID - Expression is not wrapped in `$()`.
$({body) # INVALID - Trailing curly brace is missing.
```

If a `$()` wrapper is embedded inside another `$()` wrapper, Tekton parses the contents of the innermost wrapper
as the JSONPath expression. For example:

```shell script
$($(body.b)) # Parsed as $(body.b)
$($($(body.b))) # Parsed as $(body.b)
```

## Accessing JSON keys containing periods (`.`)

To access a JSON key that contains a period (`.`), you must escape the period with a backslash (`\.`). For example:

```shell script
# Body contains a `tekton.dev` field: {"body": {"tekton.dev": "triggers"}}
$(body.tekton\.dev) -> "triggers"
```

## Fallback to default values

If Tekton fails to resolve the JSONPath expressions you have configured against the HTTP JSON payload, it
falls back to the `default` value in the corresponding `TriggerTemplate`, if specified.


## Field binding examples

Below are some examples of Tekton performing field binding based on the most commonly used field definitions:

```shell

`$(body)` -> replaced by the entire body

$(body) -> "{"key1": "value1", "key2": {"key3": "value3"}, "key4": ["value4", "value5", "value6"]}"

$(body.key1) -> "value1"

$(body.key2) -> "{"key3": "value3"}"

$(body.key2.key3) -> "value3"

$(body.key4[0]) -> "value4"

$(body.key4[0:2]) -> "{"value4", "value5"}"

# $(header) -> replaced by all headers from the event

$(header) -> "{"One":["one"], "Two":["one","two","three"]}"

$(header.One) -> "one"

$(header.one) -> "one"

$(header.Two) -> "one two three"

$(header.Two[1]) -> "two"
```

## Specifying multiple bindings

You can specify multiple bindings within the `Trigger` definition in your [`EventListener`](eventlisteners.md).
This allows you to reuse as well as mix-and-match your bindings across multiple `Trigger` definitions. For
example, you can create a `Trigger` with a binding that extracts event data and another binding that provides
information on the deployment environment:

```yaml
apiVersion: triggers.tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: event-binding
spec:
  params:
    - name: gitrevision
      value: $(body.head_commit.id)
    - name: gitrepositoryurl
      value: $(body.repository.url)
---
apiVersion: triggers.tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: prod-env
spec:
  params:
    - name: environment
      value: prod
---
apiVersion: triggers.tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: staging-env
spec:
  params:
    - name: environment
      value: staging
---
apiVersion: triggers.tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: listener
spec:
  triggers:
    - name: prod-trigger
      bindings:
        - ref: event-binding
        - ref: prod-env
      template:
        ref: pipeline-template
    - name: staging-trigger
      bindings:
        - ref: event-binding
        - ref: staging-env
      template:
        ref: pipeline-template
```

## Troubleshooting `TriggerBindings`

You can use the `binding-eval` tool to evaluate your `TriggerBinding` against a specific HTTP request
to determine the parameters that Tekton generates from that request when your corresponding `Trigger` executes.

To install the `binding-eval` tool use the following command:

```sh
$ go get -u github.com/tektoncd/triggers/cmd/binding-eval
```

Below is an example of using the tool to evaluate a `TriggerBinding`:

```sh
$ cat testdata/triggerbinding.yaml
apiVersion: tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: pipeline-binding
spec:
  params:
  - name: foo
    value: $(body.test)
  - name: bar
    value: $(header.X-Header)

$ cat testdata/http.txt
POST /foo HTTP/1.1
Content-Length: 16
Content-Type: application/json
X-Header: tacocat

{"test": "body"}

$ binding-eval -b testdata/triggerbinding.yaml -r testdata/http.txt
[
  {
    "name": "foo",
    "value": "body"
  },
  {
    "name": "bar",
    "value": "tacocat"
  }
]
```
