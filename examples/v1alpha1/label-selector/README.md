## Label Selector EventListener

Creates an EventListener that serve triggers selected via a label selector.

### Try it out locally:

1. To create the label selector trigger and all related resources, run:

   ```bash
   kubectl apply -f .
   ```

2. Port forward:
   ```bash
   kubectl port-forward service/el-label-selector-listener 8080
   ```

3. Create sample pipeline:
   ```bash
   kubectl apply -f examples/example-pipeline.yaml
   ```

3. Test by sending the sample payload.
 
   ```bash
   curl -k -v \
   -H 'X-GitHub-Event: pull_request' \
   -H 'X-Hub-Signature: sha1=8d7c4d33686fd908394208a07d997b8f5bd70aa6' \
   -H 'Content-Type: application/json' \
   -d '{"head_commit":{"id":"28911bbb5a3e2ea034daf1f6be0a822d50e31e73"},"action": "opened", "pull_request":{"head":{"sha": "28911bbb5a3e2ea034daf1f6be0a822d50e31e73"}},"repository":{"clone_url": "https://github.com/tektoncd/triggers.git", "url":"https://github.com/tektoncd/triggers.git"}}' \
   http://localhost:8080
   ```

   The response status code should be `202 Accepted`

4. You should see a single new Pipelinerun gets created, even though there are two triggers that would match the request data in the `default` namespace

   ```bash
   tkn pr list
   ```
