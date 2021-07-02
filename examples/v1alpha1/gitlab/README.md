## GitLab Push EventListener

Creates an EventListener that listens for GitLab webhook events.

### Try it out locally:

1. To create the GitLab trigger and all related resources, run:

   ```bash
   kubectl apply -f .
   ```

1. Port forward:

   ```bash
   kubectl port-forward service/el-gitlab-listener 8080
   ```

1. Test by sending the sample payload.

   ```bash
   curl -v \
   -H 'X-GitLab-Token: 1234567' \
   -H 'X-Gitlab-Event: Push Hook' \
   -H 'Content-Type: application/json' \
   --data-binary "@gitlab-push-event.json" \
   http://localhost:8080
   ```

   The response status code should be `202 Accepted`

1. You should see a new TaskRun that got created:

   ```bash
   kubectl get taskruns | grep gitlab-run-
   ```
