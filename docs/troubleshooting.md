<!--
---
linkTitle: "Troubleshooting"
---
-->
# Troubleshooting Tekton Triggers

This page describes the debugging methods you can use to diagnose and fix issues with Tekton Triggers.

## Gathering `EventListener` logs

You can gather `EventListener` logs using the Tekton `tkn` CLI tool or the Kubernetes `kubectl` CLI tool.

Use the following `tkn` command to gather `EventListener` logs:

```shell
$ tkn eventlistener logs <eventlistener-name>
```
See the `tkn` CLI tool [documentation page](https://github.com/tektoncd/cli/blob/main/docs/cmd/tkn_eventlistener_logs.md) for this config for more information.

Use the following `kubectl` command to gather `EventListener` logs:

```shell
$ kubectl logs deploy/el-<insert name of eventlistener> -n <namespace>
```

To get a list of `EventListeners` for a given namespace, use the following command:

```shell
$ kubectl get el -n <namespace>
NAME                  ADDRESS                                                        AVAILABLE   REASON
test-event-listener   http://el-test-event-listener.default.svc.cluster.local:8080   True        MinimumReplicasAvailable
```

## Configuring debug logging for `EventListeners`

`EventListeners` uses [`ConfigMap`](https://kubernetes.io/docs/concepts/configuration/configmap/) named `config-logging-triggers`
in the namespace(typically tekton-pipelines) of the target Tekton Triggers controller. To view this `ConfigMap`, use the following command: 

```shell
$ kubectl get cm -n tekton-pipelines
NAME                      DATA   AGE
config-logging-triggers   2      28m
```

Below is a typical `config-logging-listeners` `ConfigMap`:

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

To enable debug-level logging in Tekton Triggers, use the following command:

```shell
$ kubectl patch cm config-logging-triggers -n tekton-pipelines  -p '{"data": {"zap-logger-config": "{\n  \"level\": \"debug\",\n  \"development\": false,\n  \"sampling\": {\n    \"initial\": 100,\n    \"thereafter\": 100\n  },\n  \"outputPaths\": [\"stdout\"],\n  \"errorOutputPaths\": [\"stderr\"],\n  \"encoding\": \"json\",\n  \"encoderConfig\": {\n    \"timeKey\": \"ts\",\n    \"levelKey\": \"level\",\n    \"nameKey\": \"logger\",\n    \"callerKey\": \"caller\",\n    \"messageKey\": \"msg\",\n    \"stacktraceKey\": \"stacktrace\",\n    \"lineEnding\": \"\",\n    \"levelEncoder\": \"\",\n    \"timeEncoder\": \"iso8601\",\n    \"durationEncoder\": \"\",\n    \"callerEncoder\": \"\"\n  }\n}\n"}}'
```

You need to delete deployment of EventListener so that above configuration are applied to Environment variables of Deployment.
```shell
$ kubectl delete deployment/el-{EventListenerName}
```

The `EventListener` will start with the new logging level. For example:

```json
{"level":"info","ts":"2022-07-26T16:02:26.191Z","caller":"logging/config.go:116","msg":"Successfully created the logger."}
{"level":"info","ts":"2022-07-26T16:02:26.191Z","caller":"logging/config.go:117","msg":"Logging level set to: debug"}
```

From this point on, every HTTP request that the `EventListener` logs contains additional information. For example: 

```json
{"level":"debug","ts":"2021-02-10T08:42:30.915Z","logger":"eventlistener","caller":"sink/sink.go:93","msg":"EventListener: demo-event-listener in Namespace: default handling event (EventID: 9x4mb) with path /testing, payload: {\"testing\": \"value\"} and header: map[Accept:[*/*] Content-Length:[20] Content-Type:[application/x-www-form-urlencoded] User-Agent:[curl/7.61.1] X-Auth:[testing]]","knative.dev/controller":"eventlistener","/triggers-eventid":"9x4mb","logging.googleapis.com/labels":{},"logging.googleapis.com/sourceLocation":{"file":"github.com/tektoncd/triggers/pkg/sink/sink.go","line":"93","function":"github.com/tektoncd/triggers/pkg/sink.Sink.HandleEvent"}}
```

**WARNING**: The `EventListener` logs all payload headers verbatim. This includes any sensitive information the headers might contain.

To disable debug-level logging, use the following command:

```shell
$ kubectl patch cm config-logging-triggers -n <namespace> -p '{"data": {"loglevel.eventlistener": "info"}}'
```

This returns the logging level to `info`.

## Troubleshooting JSONPath issues

You may see the following message in your logs:

```json
{"level":"error","ts":"2021-02-10T08:43:47.409Z","logger":"eventlistener","caller":"sink/sink.go:230","msg":"failed to ApplyEventValuesToParams: failed to replace JSONPath value for param message: $(body.message): message is not found","knative.dev/controller":"eventlistener","/triggers-eventid":"c8f88","/trigger":"demo-trigger","logging.googleapis.com/labels":{},"logging.googleapis.com/sourceLocation":{"file":"github.com/tektoncd/triggers/pkg/sink/sink.go","line":"230","function":"github.com/tektoncd/triggers/pkg/sink.Sink.processTrigger"},"stacktrace":"github.com/tektoncd/triggers/pkg/sink.Sink.processTrigger\n\tgithub.com/tektoncd/triggers/pkg/sink/sink.go:230\ngithub.com/tektoncd/triggers/pkg/sink.Sink.HandleEvent.func1\n\tgithub.com/tektoncd/triggers/pkg/sink/sink.go:125"}
```

This means that the selected `Interceptor` is unable to parse the structure of the received JSON payload. To troubleshoot this, you must capture and inspect the payload for inconsistencies.

## Troubleshooting signature and token errors

When sending a hook protected by a secret, GitHub includes an `X-Hub-Signature` object in the header, while GitLab includes an `X-GitLab-Token` object.
You may see `no X-Hub-Signature set` and/or `no X-GitLab-Token header set` errors in your logs in one of the following scenarios:

*  If you specify a secret in your `Interceptor` but don't specify it in the hook.
*  You are sending unsigned payloads to an `Interceptor` that expects signed payloads.

Note that depending on how you have configured your Tekton Triggers stack, these errors may not indicate an actual problem. For example, your stack may
include some `Interceptors` that expect signed payloads and some that expected unsigned payloads. Since Tekton Triggers processes all `Interceptors`
concurrently, `Interceptors` that expect signed payloads will log the above errors, while `Interceptors` that expect unsigned payloads will process
those payloads successfully.
