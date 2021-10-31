# cel-eval

`cel-eval` is a tool that you can use to evaluate CEL expressions locally. It works by allowing you to define an HTTP request and a CEL expression that is evaluated against the request.

## How to use

1. Build `cel-eval` by running `go build` in this directory.
2. Define an HTTP request

```
$ cat > request <<EOF
POST /foo HTTP/1.1
Content-Length: 29
Content-Type: application/json
X-Header: tacocat

{"test": {"nested": "value"}}
EOF
```

3. Define the CEL expression

```
$ cat > expression <<EOF
body.test.nested == "value"
EOF
```

4. Run `cel-eval`

```console
$ ./cel-eval --expression ./expression --http-request ./request
true
```
