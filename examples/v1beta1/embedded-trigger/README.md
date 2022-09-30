# Triggers example

## Note that this example uses Tekton Pipeline resources, so make sure you've [installed](https://github.com/tektoncd/pipeline/blob/main/docs/install.md) that first!

In this example you will use Triggers to create a PipelineRun and
PipelineResource that simply clones a GitHub repository and prints a couple of
messages.

1. Create the resources for the example

```sh
kubectl apply -f rbac.yaml
kubectl apply -f triggertemplate.yaml
kubectl apply -f triggerbinding.yaml
kubectl apply -f triggerbinding-message.yaml
kubectl apply -f eventlistener.yaml
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/main/task/git-clone/0.8/git-clone.yaml
```

2. Check required pods and services are available and healthy

```bash
tekton:examples user$ kubectl get svc
NAME                          TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)          AGE
el-embedded-trigger-listener  ClusterIP      10.100.151.220   <none>        8080/TCP,9000/TCP         48s  <--- this will receive the event
```

```bash
tekton:examples user$ kubectl get pods
NAME                                           READY     STATUS    RESTARTS   AGE
el-listener-5c744f47c5-m9kdn                   1/1       Running   0          78s
```

3. Apply an example pipeline and tasks that will be run (in this case named
   `example-pipeline`):

```bash
kubectl apply -f ../../example-pipeline.yaml
```

This is intentionally very simple and operates on a Git workspace. The
trigger created Git workspace will clone from the repository URL and revision
parameters.

4. Send a payload to the listener

Assuming we have a listener available at `localhost:8080` (and port-forwarded
for this example, with
`kubectl port-forward service/el-embedded-trigger-listener 8080`),
run the following command in your shell of choice or using Postman:

```bash
curl -X POST \
  http://localhost:8080 \
  -H 'Content-Type: application/json' \
  -H 'X-Hub-Signature: sha1=2da37dcb9404ff17b714ee7a505c384758ddeb7b' \
  -d '{
	"repository":
	{
		"url": "https://github.com/tektoncd/triggers.git"
	}
}'
```

NOTE: defaults in `triggertemplate.yaml` like `main` for `gitrevision` are leveraged here to 
satisfy missing items in the POST body like `head_commit.id`.

5. Observe created PipelineRun

```bash
tekton:examples user$ kubectl get pipelinerun
NAME                       SUCCEEDED   REASON    STARTTIME   COMPLETIONTIME
simple-pipeline-run-n4qps   Unknown     Running   1s
```

```bash
tekton:examples user$ kubectl get pods
...
simple-pipeline-run-n4qps-say-hello-djs4v-pod-64cfef   0/2       Init:0/2   0          1s
...
```

# What just happened?

1. A `PipelineRun` with an embedded `resourceSpec` was created for us using our
   POST data and the specified Tekton Pipeline:

```yaml
---
spec:
  params:
    - name: message
      value: Hello from the Triggers EventListener!
    - name: contenttype
      value: application/json
    - name: git-revision
      value: main
    - name: git-url
      value: https://github.com/tektoncd/triggers.git
  pipelineRef:
    name: simple-pipeline
  workspaces:
    - name: git-source
      emptyDir: {}
  timeout: 1h0m0s
```

2. The four Pods (one per Task) finish their work and the PipelineRun is marked
   as successful:

```
tekton:examples user$ logs simple-pipeline-run-n4qps-git-clone-pod --all-containers
2022/09/30 14:11:20 Entrypoint initialization
2022/09/30 14:11:21 Decoded script /tekton/scripts/script-0-xvr6x
+ '[' false '=' true ]
+ '[' false '=' true ]
+ '[' false '=' true ]
+ CHECKOUT_DIR=/workspace/output/
+ '[' true '=' true ]
+ cleandir
+ '[' -d /workspace/output/ ]
+ rm -rf '/workspace/output//*'
+ rm -rf '/workspace/output//.[!.]*'
+ rm -rf '/workspace/output//..?*'
+ test -z 
+ test -z 
+ test -z 
+ /ko-app/git-init '-url=https://github.com/tektoncd/triggers.git' '-revision=main' '-refspec=' '-path=/workspace/output/' '-sslVerify=true' '-submodules=true' '-depth=1' '-sparseCheckoutDirectories='
{"level":"info","ts":1664547088.5813992,"caller":"git/git.go:170","msg":"Successfully cloned https://github.com/tektoncd/triggers.git @ c147f7d9766f33039791af401a3404c005db1459 (grafted, HEAD, origin/main) in path /workspace/output/"}
{"level":"info","ts":1664547088.5921903,"caller":"git/git.go:208","msg":"Successfully initialized and updated submodules in path /workspace/output/"}
+ cd /workspace/output/
+ git rev-parse HEAD
+ RESULT_SHA=c147f7d9766f33039791af401a3404c005db1459
+ EXIT_CODE=0
+ '[' 0 '!=' 0 ]
+ printf '%s' c147f7d9766f33039791af401a3404c005db1459
+ printf '%s' https://github.com/tektoncd/triggers.git

tekton:examples user$ kubectl logs simple-pipeline-run-n4qps-say-hello-29ztk-pod-118fbd --all-containers
...
Hello Triggers!
```

```
tekton:examples user$ kubectl logs simple-pipeline-run-n4qps-say-message-f64qf-pod-80fb58 --all-containers
...
Hello from the Triggers EventListener!
```

```
tekton:examples user$ kubectl logs simple-pipeline-run-n4qps-say-bye-7xbk2-pod-116608  --all-containers
...
Goodbye Triggers!
```

```
tekton:examples user$ kubectl get pipelinerun
NAME                       SUCCEEDED   REASON      STARTTIME   COMPLETIONTIME
simple-pipeline-run-n4qps   True        Succeeded   5m          4m
```

# Cleaning up

```sh
kubectl delete pipelinerun -l triggers.tekton.dev/eventlistener=embedded-trigger-listener
```

# Conclusion

We hope you've found this example useful, please do get involved and contribute
more useful examples!
