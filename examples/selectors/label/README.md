## Label Selector EventListener

Creates an EventListener that serve triggers selected via a label selector.

### Try it out locally:

1. To create the label selector trigger and all related resources, run:

   ```bash
   kubectl apply -f examples/selectors/label
   ```

2. Port forward:
   ```bash
   kubectl port-forward \
   -n foo $(kubectl get pod -n foo -o=name \
   -l eventlistener=listener-label-selector) 8000
   ```

   **Note**: Instead of port forwarding, you can set the
   [`serviceType`](https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#serviceType)
   to `LoadBalancer` to expose the EventListener with a public IP. 
   For this example, modify `02_eventlistener-label-sel.yaml`

3. Create sample pipeline in namespace `foo`:
   ```bash
   kubectl apply -f examples/example-pipeline.yaml -n foo
   ```

3. Test by sending the sample payload.

   ```bash
       curl -k -v \
       -H 'X-GitHub-Event: pull_request' \
       -H 'X-Hub-Signature: sha1=8d7c4d33686fd908394208a07d997b8f5bd70aa6' \
       -H 'Content-Type: application/json' \
       -d '{"head_commit":{"id":"28911bbb5a3e2ea034daf1f6be0a822d50e31e73"},"action": "opened", "pull_request":{"head":{"sha": "28911bbb5a3e2ea034daf1f6be0a822d50e31e73"}},"repository":{"clone_url": "https://github.com/tektoncd/triggers.git", "url":"https://github.com/tektoncd/triggers.git"}}' http://localhost:8000   ```

   The response status code should be `201 Created`

4. You should see a single new Pipelinerun gets created, even though there are two triggers that would match the request data in the `foo` namespace

   ```bash
   tkn pr -n foo list
   ```
