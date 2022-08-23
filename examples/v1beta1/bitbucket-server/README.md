## Bitbucket EventListener

Creates an EventListener that listens for Bitbucket server webhook events.

### Try it out locally:

1. To create the Bitbucket trigger and all related resources, run:

   ```bash
   kubectl apply -f .
   ```

1. Port forward:

   ```bash
   kubectl port-forward service/el-bitbucket-server-listener 8080
   ```

1. Test by sending the sample payload.

   ```bash
   curl -v \
   -H 'X-Event-Key: repo:refs_changed' \
   -H 'X-Hub-Signature: sha1=544dde67b85743361b0321c9cc55f84472a608dd' \
   -d '{"repository": {"links": {"clone": [{"href": "http://localhost:7990/scm/~test/helloworld.git", "name": "http"}, {"href": "ssh://git@localhost:7999/~test/helloworld.git", "name": "ssh"}]}}, "changes": [{"ref": {"displayId": "main"}}]}' \
   http://localhost:8080
   ```

   The response status code should be `202 Accepted`

   [`HMAC`](https://www.freeformatter.com/hmac-generator.html) tool used to create X-Hub-Signature.

   In [`HMAC`](https://www.freeformatter.com/hmac-generator.html) `string` is the *body payload ex:* `{"repository": {"links": {"clone": [{"href": "http://localhost:7990/scm/~test/helloworld.git", "name": "http"}, {"href": "ssh://git@localhost:7999/~test/helloworld.git", "name": "ssh"}]}}, "changes": [{"ref": {"displayId": "main"}}]}`
   and `secretKey` is the *given secretToken ex:* `1234567`.

1. You should see a new TaskRun that got created:

   ```bash
   kubectl get taskruns | grep bitbucket-run-
   ```
