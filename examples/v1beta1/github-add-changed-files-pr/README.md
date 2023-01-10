## GitHub EventListener

Creates an EventListener that listens for GitHub webhook events and adds the files that have changed within the pull request or push to the github payload. The list of changed files are added to the `changed_files` property of the event payload in the top-level `extensions` field

### Try it out locally:

1. To create the GitHub trigger and all related resources, run:

   ```bash
   kubectl apply -f .
   ```

1. Port forward:

   ```bash
   kubectl port-forward service/el-github-add-changed-files-pr-listener 8080
   ```

1. Test by sending the sample payload.

   ```bash
    curl -v \
    -H 'X-GitHub-Event: pull_request' \
    -H 'Content-Type: application/json' \
    -d '{"action": "opened","number": 1503,"pull_request": {"head": {"sha": "16dd484bb4888dd30154f5ccb765beae1aaf72de"}},"repository": {"full_name": "tektoncd/triggers","clone_url": "https://github.com/tektoncd/triggers.git"}}' \
    http://localhost:8080
   ```

   The response status code should be `202 Accepted`

   [`HMAC`](https://www.freeformatter.com/hmac-generator.html) tool used to create X-Hub-Signature.

   In [`HMAC`](https://www.freeformatter.com/hmac-generator.html) `string` is the *body payload ex:* `{"action": "opened", "pull_request":{"head":{"sha": "28911bbb5a3e2ea034daf1f6be0a822d50e31e73"}},"repository":{"clone_url": "https://github.com/tektoncd/triggers.git"}}`
   and `secretKey` is the *given secretToken ex:* `1234567`.

1. You should see a new TaskRun that got created:

   ```bash
   kubectl get taskruns | github-add-changed-files-pr-run-
   ```

1. Get the pod created from the TaskRun and show the logs to see the changed files:

   ```bash
   kubectl get pods | grep github-add-changed-files-pr-run-
   kubectl logs <POD NAME>
   ```
