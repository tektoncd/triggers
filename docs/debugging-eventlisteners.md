<!--
---
linkTitle: "Debugging EventListeners"
---
-->
# EventListener logs

From the command-line, the easiest way to get the logs is:

```shell
$ tkn eventlistener logs
```
See [here](https://github.com/tektoncd/cli/blob/master/docs/cmd/tkn_eventlistener_logs.md) for more uses of the config.

Alternatively, you can get them with kubectl:

```shell
$ kubectl logs deploy/el-<insert name of eventlistener> -n <namespace>
```
You can get a list of EventListeners in a namespace with:

```shell
$ kubectl get el -n <namespace>
NAME                  ADDRESS                                                        AVAILABLE   REASON
test-event-listener   http://el-test-event-listener.default.svc.cluster.local:8080   True        MinimumReplicasAvailable
```

## Enable Debug Logging

Triggers creates a Kubernetes [ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/) in the namespace that the EventListener
is created in called `config-logging-triggers`.

You can see this with:

```shell
$ kubectl get cm -n <namespace>
NAME                      DATA   AGE
config-logging-triggers   2      28m
```

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-logging-triggers
data:
  loglevel.eventlistener: info
  zap-logger-config: '{"level": "info","development": false,"sampling": {"initial":
    100,"thereafter": 100},"outputPaths": ["stdout"],"errorOutputPaths": ["stderr"],"encoding":
    "json","encoderConfig": {"timeKey": "ts","levelKey": "level","nameKey": "logger","callerKey":
    "caller","messageKey": "msg","stacktraceKey": "stacktrace","lineEnding": "","levelEncoder":
    "","timeEncoder": "iso8601","durationEncoder": "","callerEncoder": ""}}'
```

### Enable debug level logging

The simplest way to enable additional debug logging is:

```shell
$ kubectl patch cm config-logging-triggers -n <namespace> -p '{"data": {"loglevel.eventlistener": "debug"}}'
```

The EventListener should log out that it has changed the log level:

```json
{"level":"info","ts":"2021-02-10T08:42:10.950Z","logger":"eventlistener","caller":"logging/config.go:207","msg":"Updating logging level for eventlistener from debug to info.","knative.dev/controller":"eventlistener","logging.googleapis.com/labels":{},"logging.googleapis.com/sourceLocation":{"file":"knative.dev/pkg@v0.0.0-20210107022335-51c72e24c179/logging/config.go","line":"207","function":"knative.dev/pkg/logging.UpdateLevelFromConfigMap.func1"}}
```

When `debug` level is enabled, each HTTP request that the EventListener receives
will be logged with additional information:

```json
{"level":"debug","ts":"2021-02-10T08:42:30.915Z","logger":"eventlistener","caller":"sink/sink.go:93","msg":"EventListener: demo-event-listener in Namespace: default handling event (EventID: 9x4mb) with path /testing, payload: {\"testing\": \"value\"} and header: map[Accept:[*/*] Content-Length:[20] Content-Type:[application/x-www-form-urlencoded] User-Agent:[curl/7.61.1] X-Auth:[testing]]","knative.dev/controller":"eventlistener","/triggers-eventid":"9x4mb","logging.googleapis.com/labels":{},"logging.googleapis.com/sourceLocation":{"file":"github.com/tektoncd/triggers/pkg/sink/sink.go","line":"93","function":"github.com/tektoncd/triggers/pkg/sink.Sink.HandleEvent"}}
```
**WARNING**: The headers are logged out verbatim, if your client is sending any sort of sensitive data in the headers, these will be logged out.

### Return to info level debugging

You can restore it to the default logging level `info` with:

```shell
$ kubectl patch cm config-logging-triggers -n <namespace> -p '{"data": {"loglevel.eventlistener": "info"}}'
```

## Debugging JSONPath issues

```json
A{"level":"error","ts":"2021-02-10T08:43:47.409Z","logger":"eventlistener","caller":"sink/sink.go:230","msg":"failed to ApplyEventValuesToParams: failed to replace JSONPath value for param message: $(body.message): message is not found","knative.dev/controller":"eventlistener","/triggers-eventid":"c8f88","/trigger":"demo-trigger","logging.googleapis.com/labels":{},"logging.googleapis.com/sourceLocation":{"file":"github.com/tektoncd/triggers/pkg/sink/sink.go","line":"230","function":"github.com/tektoncd/triggers/pkg/sink.Sink.processTrigger"},"stacktrace":"github.com/tektoncd/triggers/pkg/sink.Sink.processTrigger\n\tgithub.com/tektoncd/triggers/pkg/sink/sink.go:230\ngithub.com/tektoncd/triggers/pkg/sink.Sink.HandleEvent.func1\n\tgithub.com/tektoncd/triggers/pkg/sink/sink.go:125"}
```
This means that the JSON body received by the interceptor likely doesn't have
the structure you expect, you will probably need to capture the request somehow
and inspect it.

## Debugging 'no X-Hub-Signature set' and 'no X-GitLab-Token header set'

GitHub sends a header `X-Hub-Signature` when it's sending a hook that is
configured with a secret, and GitLab sends the `X-GitLab-Token` header.

This error can occur in two different ways, if you set a secret in the GitHub or
GitLab interceptor, but don't configure the corresponding hook with a secret,
you will trigger these errors.

Alternatively this can occur when you have unsigned requests being routed
through an interceptor that requires them.

This can be a side-effect of your Trigger's interceptor stack, as all
interceptors are processed concurrently, so triggers that have a GitHub
interceptor will log this out, even though another trigger might successfully
process the request.
