## GitHub EventListener

Creates an EventListener that listens for GitHub webhook events.

### Try it out locally:

1. To create the GitHub trigger and all related resources, run:

   ```bash
   kubectl apply -f examples/github/
   ```

1. Port forward:

   ```bash
   kubectl port-forward \
    "$(kubectl get pod --selector=eventlistener=github-listener-interceptor -oname)" \
     8080
   ```

   **Note**: Instead of port forwarding, you can set the
   [`serviceType`](https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#serviceType)
   to `LoadBalancer` to expose the EventListener with a public IP.

1. Test by sending the sample payload.

   ```bash
   curl -v \
   -H 'X-GitHub-Event: pull_request' \
   -H 'X-Hub-Signature: sha1=13eaa0168f8d8efcdf5189ea75b782cf89809de6' \
   -H 'Content-Type: application/json' \
   -d '{"action": "opened", "head_commit":{"id":"master"},"repository":{"url": "https://github.com/tektoncd/triggers"}}' \
   http://localhost:8080
   ```

   The response status code should be `201 Created`
   
   [`HMAC`](https://www.freeformatter.com/hmac-generator.html) tool used to create X-Hub-Signature. 
   
   In [`HMAC`](https://www.freeformatter.com/hmac-generator.html) `string` is the *body payload* and `secretKey` is the *given secretToken*.

1. You should see a new TaskRun that got created:

   ```bash
   kubectl get taskruns | grep github-run-
   ```
