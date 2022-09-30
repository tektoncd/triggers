# Triggers Examples


This folder contains a number of examples of running Triggers with various configurations. The v1alpha1 folder contains
examples that use the v1alpha1 version of the Triggers while the v1beta1 folder contains resources using v1beta1 versions
of Triggers resources. Many of the sub-folders also contain their own READMEs with information specific to the example.

# Running the Examples

## Pre-Requisites
To run the examples, you need the following pre-requisites:

1. Ensure you have Tekton Pipelines [installed](https://github.com/tektoncd/pipeline/blob/master/docs/install.md)

2. Create the service account and all associated roles and bindings by running `kubectl apply -f rbac.yaml`.

**Note**: `rbac.yaml` assumes that examples are running in the default namespace. If you would like to run examples
in a different namespace, edit the `triggers-example-eventlistener-clusterbinding` ClusterRoleBinding to refer to
the namespace where you've deployed the service account, for example:

```yaml
subjects:
- kind: ServiceAccount
  name: tekton-triggers-example-sa
  namespace: my-favorite-namespace
```

3. Apply the `git-clone` task from the Tekton catalog to the cluster

This can be done either via kubectl:

```
kubectl apply -n my-favorite-namespace -f https://raw.githubusercontent.com/tektoncd/catalog/main/task/git-clone/0.8/git-clone.yaml
```

or via the `tkn` CLI:

```
tkn hub install task git-clone -n my-favorite-namespace
```

## Creating Triggers Resources

Create the trigger resources for each example by applying the YAMLs from the sub-folders.  Some examples have 
their own READMEs with further instructions.
   
At this point, you can invoke the Trigger locally to test it out.

## Invoking the Triggers locally

1. Access the EventListener locally by port-forwarding to it. Each EventListener exposes a service with the same name 
   as the EventListener but prefixed with the string `el-`:

```bash
EVENTLISTENER_NAME=example-listener
kubectl port-forward service/el-${EVENTLISTENER_NAME} 8080
```

2. Once the port-forward is done, you can invoke the Trigger by make an HTTP request to `localhost:8080` using `curl`.
The HTTP request must be a POST request that contains a JSON payload. The JSON payload should contain any fields that they are referenced via a TriggerBinding within the Trigger. 
   For example, for a Trigger that contains a binding with the value `$(body.commit_sha)`, the payload should contain a field called `commit_sha`.

   
```bash
curl -X POST \
  http://localhost:8080 \
  -H 'Content-Type: application/json' \
  -d '{ "commit_sha": "22ac84e04fd2bd9dce8529c9109d5bfd61678b29" }'
```

You should expect to get a response back with a `202 Accepted` status code. The response should contain a JSON body that 
contains a `eventID` field. 

```json
{
  "eventListener":"example-listener",
  "namespace":"default",
  "eventListenerUID":"83f2cce3-12e2-4153-997e-4c16ac0dbe5b",
  "eventID":"acc66b28-bdc0-4099-b51c-ad475bcc4c2b"
}
```

3. The EventListener will attach the `eventID` to any Tekton resources it creates. So, we can use it to query for any 
   resources that were created:

```bash
eventID=
kubectl get taskruns -l triggers.tekton.dev/triggers-eventid=${eventID}
```


In case you do not see any resources that have been created, you can use the eventID to search the logs for 
the EventListener:

```bash
EVENTLISTENER_NAME=example-listener
kubectl logs deployments/el-${EVENTLISTENER_NAME}
```
