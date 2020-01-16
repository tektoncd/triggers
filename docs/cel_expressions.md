# CEL expression extensions

The CEL expression is configured to expose parts of the request, and some custom
functions to make matching easier.

In addition to the custom function extension listed below, you can craft any
valid CEL expression as defined by the
[cel-spec language definition](https://github.com/google/cel-spec/blob/master/doc/langdef.md)

### List of extensions

The body from the `http.Request` value is decoded to JSON and exposed, and the
headers are also available.

<table style="width=100%" border="1">
  <tr>
    <th>Symbol</th>
    <th>Type</th>
    <th>Description</th>
    <th>Example</th>
  </tr>
  <tr>
    <th>
      body
    </th>
    <td>
      map(string, dynamic)
    </td>
    <td>
      This is the decoded JSON body from the incoming http.Request exposed as a map of string keys to any value types.
    </td>
    <td>
      <pre>body.value == 'test'</pre>
    </td>
  </tr>
  <tr>
    <th>
      header
    </th>
    <td>
      map(string, list(string))
    </td>
    <td>
      This is the request Header.
    </td>
    <td>
      <pre>header['X-Test'][0] == 'test-value'</pre>
    </td>
  </tr>
</table>

NOTE: The header value is a Go `http.Header`, which is
[defined](https://golang.org/pkg/net/http/#Header) as:

```go
type Header map[string][]string
```

i.e. the header is a mapping of strings, to arrays of strings, see the `match`
function on headers below for an extension that makes looking up headers easier.

### List of extension functions

This lists custom functions that can be used from CEL expressions in the CEL
interceptor.

<table style="width=100%" border="1">
  <tr>
    <th>Symbol</th>
    <th>Type</th>
    <th>Description</th>
    <th>Example</th>
  </tr>
  <tr>
    <th>
      match
    </th>
    <td>
      header.(string, string) -> bool
    </td>
    <td>
      Uses the canonical header matching from Go's http.Request to match the header against the value.
    </td>
    <td>
     <pre>header.match('x-test', 'test-value')</pre>
    </td>
  </tr>
  <tr>
    <th>
      truncate
    </th>
    <td>
      (string, uint) -> string
    </td>
    <td>
      Truncates a string to no more than the specified length.
    </td>
    <td>
     <pre>truncate(body.commit.sha, 5)</pre>
    </td>
  </tr>
  <tr>
    <th>
      split
    </th>
    <td>
      (string, string) -> string(dyn)
    </td>
    <td>
      Splits a string on the provided separator value.
    </td>
    <td>
     <pre>split(body.ref, '/')</pre>
    </td>
  </tr>
    <th>
      canonical
    </th>
    <td>
      header.(string) -> string
    </td>
    <td>
      Uses the canonical header matching from Go's http.Request to get the provided header name.
    </td>
    <td>
     <pre>header.canonical('x-test')</pre>
    </td>
  </tr>
  <tr>
    <th>
      decodeb64
    </th>
    <td>
      (string) -> string
    </td>
    <td>
      Decodes a base64 encoded string.
    </td>
    <td>
     <pre>decodeb64(body.message.data)</pre>
    </td>
  </tr>
</table>
