## Namespace Selector EventListener

Creates an EventListener that serve triggers in multiple namespaces.

### Try it out locally:

1. To create the namespace selector trigger and all related resources, run:

   ```bash
   kubectl apply -f examples/selectors/
   ```

2. Port forward:

   ```bash
       kubectl config set-context --current --namespace=bar
       kubectl apply -f examples/example-pipeline.yaml
   ```

   **Note**: Instead of port forwarding, you can set the
   [`serviceType`](https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#serviceType)
   to `LoadBalancer` to expose the EventListener with a public IP.

3. Create sample pipelinerun in namespace bar:
   ```bash
   kubectl port-forward \
   -n foo $(kubectl get pod -n foo -o=name \
   -l eventlistener=listener-sel) 8080
   ```

3. Test by sending the sample payload.

   ```bash
       curl -k -v \
       -H 'X-GitHub-Event: pull_request' \
       -H 'X-Hub-Signature: sha1=8d7c4d33686fd908394208a07d997b8f5bd70aa6' \
       -H 'Content-Type: application/json' \
       -d '{"head_commit":{"id":"28911bbb5a3e2ea034daf1f6be0a822d50e31e73"},"action": "opened", "pull_request":{"head":{"sha": "28911bbb5a3e2ea034daf1f6be0a822d50e31e73"}},"repository":{"clone_url": "https://github.com/tektoncd/triggers.git", "url":"https://github.com/tektoncd/triggers.git"}}' http://localhost:8080   ```

   The response status code should be `201 Created`

4. You should see a new Pipelinerun that got created:

   ```bash
   tkn pr -n bar list
   ```
