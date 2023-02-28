## GitHub Owner EventListener

Creates an EventListener that listens for `pull_request` or `issue_comment` GitHub webhook events and will only continue processing if the user making the request is listed in the [`owners` file](https://www.kubernetes.dev/docs/guide/owners/)

### Try it out locally:

1. To create the GitHub trigger and all related resources, run:

   ```bash
   kubectl apply -f .
   ```

1. Port forward:

   ```bash
   kubectl port-forward service/el-github-owners-listener 8080
   ```

1. Test by sending the sample payload.

   ```bash
   curl -v \
   -H 'X-GitHub-Event: pull_request' \
   -H 'X-Hub-Signature: sha1=70d0ebf86a7973374898b7711acc0897616e2c93' \
   -H 'Content-Type: application/json' \
   -d '{"action": "opened","number": 1503,"repository":{"full_name": "tektoncd/triggers", "clone_url": "https://github.com/tektoncd/triggers.git"}, "sender":{"login": "dibyom"}}' \
   http://localhost:8080
   ```

   The response status code should be `202 Accepted`

   [`HMAC`](https://www.freeformatter.com/hmac-generator.html) tool used to create X-Hub-Signature.

   In [`HMAC`](https://www.freeformatter.com/hmac-generator.html) `string` is the *body payload ex:* `{"action": "opened","number": 1503,"repository":{"full_name": "tektoncd/triggers", "clone_url": "https://github.com/tektoncd/triggers.git"}, "sender":{"login": "dibyom"}}`
   and `secretKey` is the *given secretToken ex:* `1234567`.

1. You should see a new TaskRun that got created:

   ```bash
   kubectl get taskruns | grep github-run-
   ```
