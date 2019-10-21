# TriggerBindings
As per the name, `TriggerBinding`s bind against events/triggers.
`TriggerBinding`s enable you to capture fields from an event and store
them as parameters. The separation of `TriggerBinding`s from `TriggerTemplate`s
was deliberate to encourage reuse between them.

<!-- FILE: examples/triggerbindings/triggerbinding.yaml -->
```YAML
apiVersion: tekton.dev/v1alpha1
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

`TriggerBinding`s are connected to `TriggerTemplate`s within an [`EventListener`](eventlisteners.md), which is where the pod is actually instantiated that "listens" for the respective events.

## Parameters
`TriggerBinding`s can provide `params` which are passed to a
`TriggerTemplate`. Each parameter has a `name` and a `value`.


## Event Variable Interpolation

In order to parse generic events as efficiently as possible, [GJSON](https://github.com/tidwall/gjson)
is used internally. As a result, the binding [path syntax](https://github.com/tidwall/gjson#path-syntax)
differs slightly from standard JSON. As of now, the following patterns are
supported within `TriggerBinding` parameter value interpolation:
- `\$\(body(\.[[:alnum:]/_\-\.\\]+|\.#\([[:alnum:]=<>%!"\*_-]+\)#??)*\)`
- `\$\(header(\.[[:alnum:]_\-]+)?\)`

### Body
HTTP Post request body data can be referenced using variable interpolation.
Text in the form of `$(body.X.Y.Z)` is replaced by the body data at JSON path
`X.Y.Z`.

`$(body)` is replaced by the entire body.

The following are some example variable interpolation replacements:
```
$(body) -> "{\"key1\": \"value1\", \"key2\": {\"key3\": \"value3\"}, \"key4\": [\"value4\", \"value5\"]}"

$(body.key1) -> "value1"

$(body.key2) -> "{\"key3\": \"value3\"}"

$(body.key2.key3) -> "value3"

$(body.key4.0) -> "value4"
```

### Header
HTTP Post request header data can be referenced using variable interpolation.
Text in the form of `$(header.X)` is replaced by the event's header named `X`.

`$(header)` is replaced by all of the headers from the event.

The following are some example variable interpolation replacements:
```
$(header) -> "{\"One\":[\"one\"], \"Two\":[\"one\",\"two\",\"three\"]}"

$(header.One) -> "one"

$(header.Two) -> "one two three"

$(header.Two.1) -> "two"
```
