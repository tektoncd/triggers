## Bitbucket EventListener

Creates an EventListener that listens for Bitbucket webhook events.

### Try it out locally:

1. To create the Bitbucket cloud trigger and all related resources, run:

   ```bash
   kubectl apply -f .
   ```

1. Port forward:

   ```bash
   kubectl port-forward service/el-bitbucket-cloud-listener 8080
   ```

1. Test by sending the sample payload.

   ```bash
   curl -v \
   -H 'X-Event-Key: repo:push' \
   -d '{"push": {"changes":[{"old":{}, "new":{"name":"bb-cloud-test-1"}}]},"repository":{"links": {"html":{"href": "https://bitbucket.org/savitaredhat/bitbuckettest-customer"}},"name":"bitbuckettest-customer"}, "actor":{"display_name": "savita"}}' \
   http://localhost:8080
   ```

   The response status code should be `202 Accepted`

1. You should see a new TaskRun that got created:

   ```bash
   kubectl get taskruns | grep bitbucket-cloud-run-
   ```
