## GitHub Knative EventListener

Creates an EventListener that listens for GitHub webhook events.

### Try it out locally:

1. To create the custom resource trigger and all related resources, run:

   ```bash
   kubectl apply -f .
   ```

1. Test by sending the sample payload:

   ```bash
   curl -v \
   -H 'X-GitHub-Event: pull_request' \
   -H 'X-Hub-Signature: sha1=ba0cdc263b3492a74b601d240c27efe81c4720cb' \
   -H 'Content-Type: application/json' \
   -d '{"action": "opened", "pull_request":{"head":{"sha": "28911bbb5a3e2ea034daf1f6be0a822d50e31e73"}},"repository":{"clone_url": "https://github.com/tektoncd/triggers.git"}}' \
   http://<el_address>
   ```

   The response status code should be `202 Accepted`
   
   [`HMAC`](https://www.freeformatter.com/hmac-generator.html) tool used to create X-Hub-Signature.
   
   In [`HMAC`](https://www.freeformatter.com/hmac-generator.html) `string` is the *body payload ex:* `{"action": "opened", "pull_request":{"head":{"sha": "28911bbb5a3e2ea034daf1f6be0a822d50e31e73"}},"repository":{"clone_url": "https://github.com/tektoncd/triggers.git"}}`
   and `secretKey` is the *given secretToken ex:* `1234567`.

1. You will see the newly created TaskRun:

   ```bash
   kubectl get taskruns | grep github-run-
   ```
