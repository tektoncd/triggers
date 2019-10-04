## Sample Event Interceptors

This folder contains some sample [event interceptors](../../docs/eventlisteners.md#event-interceptors):

* [Github Validation](#github-validation)


### GitHub Validation

Validates Github webhook payloads using a supplied webhook secret token. Assumes 
that a secret called `github-secret` is present and that it contains a key 
called `secret-token`. Also assumes that a Github webhook has been configured 
with this secret token.

To create the secret:

```yaml
kubectl create secret generic github-secret-2 --from-literal=secret-token="$(echo <REPLACE_WITH_SECRET_STRING>|base64)"
```

To deploy the service, you can either: 

* Use `ko` to build and deploy from source:

```yaml
ko apply -f examples/event-interceptors/github-validate.yaml
```

* Use the [create-gh-validate-interceptor](../../docs/create-gh-validate-interceptor.yaml) Tekton task to deploy the latest released version of the container image

#### Releasing a new version of the image

**Note**: This is temporary till #155 is done

```yaml

KO_DOCKER_REPO=gcr.io/tekton-releases/triggers ko publish --preserve-import-paths github.com/tektoncd/triggers/cmd/gh-validate

```
