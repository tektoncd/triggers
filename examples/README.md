# Triggers example

## Note that this example uses Tekton Pipeline resources, so make sure you've [installed](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) that first!

In this example you will use Triggers to create a PipelineRun and PipelineResource that simply clones a GitHub repository and prints a couple of messages.

1. Create the resources for the example

```yaml
kubectl apply -f role-resources
kubectl apply -f triggertemplates/triggertemplate.yaml
kubectl apply -f triggerbindings/triggerbinding.yaml
kubectl apply -f eventlisteners/eventlistener.yaml
```

2. Check required pods and services are available and healthy

```bash
tekton:examples user$ kubectl get svc
NAME                          TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)          AGE
listener                      LoadBalancer   10.100.151.220   localhost     8082:30607/TCP   48s  <--- this will receive the event
tekton-pipelines-controller   ClusterIP      10.103.144.96    <none>        9090/TCP         8m34s
tekton-pipelines-webhook      ClusterIP      10.96.198.4      <none>        443/TCP          8m34s
tekton-triggers-controller    ClusterIP      10.102.221.96    <none>        9090/TCP         7m56s
tekton-triggers-webhook       ClusterIP      10.99.59.231     <none>        443/TCP          7m56s
```

```bash
tekton:examples user$ kubectl get pods
NAME                                           READY     STATUS    RESTARTS   AGE
listener-5c744f47c5-m9kdn                      1/1       Running   0          78s
tekton-pipelines-controller-55c6b5b9f6-qsdnn   1/1       Running   0          9m4s
tekton-pipelines-webhook-6794d5bcc8-p4p8c      1/1       Running   0          9m4s
tekton-triggers-controller-594d4fcfdf-l4c9m    1/1       Running   0          6m57s
tekton-triggers-webhook-5985cfcfc5-cq5hp       1/1       Running   0          6m50s
```

3. Apply an example pipeline and tasks that will be run (in this case named `simple-pipeline`):

`kubectl apply -f example-pipeline.yaml`

This is intentionally very simple and operates on a created Git resource. The trigger created Git resource will have the repository URL and revision parameters.

4. Send a payload to the listener

Assuming we have a listener available at `localhost:8082` (and port-forwarded for this example, with `kubectl port-forward $(kubectl get pod -o=name -l app=listener) 8082`), run the following command in your shell of choice or using Postman:

```bash
curl -X POST \
  http://localhost:8082 \
  -H 'Content-Type: application/json' \
  -d '{
	"head_commit":
	{
		"id": "master"
	}, 
	"repository": 
	{
		"url": "https://github.com/tektoncd/triggers.git"
	}
}'
```

5. Observe created PipelineRun and PipelineResources

```bash
tekton:examples user$ kubectl get pipelineresources
NAME               AGE
git-source-g8j7r   1s
```

```bash
tekton:examples user$ kubectl get pipelinerun
NAME                       SUCCEEDED   REASON    STARTTIME   COMPLETIONTIME
simple-pipeline-runxl8rm   Unknown     Running   1s      
```

```bash
tekton:examples user$ kubectl get pods
...
simple-pipeline-runnd654-say-hello-djs4v-pod-64cfef   0/2       Init:0/2   0          1s
...
```

# What just happened?

1. A `PipelineResource` was created for us: notice the parameters matching with our POST data.

```
tekton:examples user$ kubectl get pipelineresource git-source-g8j7r  -o yaml
apiVersion: tekton.dev/v1alpha1
kind: PipelineResource
metadata:
  labels:
    triggertemplated: "true"
  name: git-source-g8j7r
  namespace: tekton-pipelines
...
spec:
  params:
  - name: revision
    value: master
  - name: url
    value: https://github.com/tektoncd/triggers.git
  type: git
```

2. A `PipelineRun` was created using this resource and the specified Tekton Pipeline:

```
spec:
  pipelineRef:
    name: simple-pipeline
  podTemplate: {}
  resources:
  - name: git-source
    resourceRef:
      name: git-source-g8j7r
  serviceAccount: ""
  timeout: 1h0m0s
status:
  completionTime: 2019-09-12T12:46:44Z
  conditions:
  - lastTransitionTime: 2019-09-12T12:46:44Z
    message: All Tasks have completed executing
    reason: Succeeded
    status: "True"
    type: Succeeded
  ...
```

3. The two Pods (one per Task) finish their work and the PipelineRun is marked as successful:

```
tekton:examples user$ kubectl logs simple-pipeline-runn4qps-say-hello-29ztk-pod-118fbd --all-containers
...
Hello Triggers!
```

```
tekton:examples user$ kubectl logs simple-pipeline-runn4qps-say-bye-7xbk2-pod-116608  --all-containers
...
Goodbye Triggers!
```

```
tekton:examples user$ kubectl get pipelinerun
NAME                       SUCCEEDED   REASON      STARTTIME   COMPLETIONTIME
simple-pipeline-runn4qps   True        Succeeded   5m          4m
```

# Cleaning up

```yaml
kubectl delete all -l generatedBy=triggers-example
```

# Conclusion

We hope you've found this example useful, please do get involved and contribute more useful examples!