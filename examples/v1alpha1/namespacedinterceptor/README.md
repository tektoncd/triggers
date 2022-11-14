## NamespacedInterceptor

Creates a NamespacedInterceptor and an EventListener which utilizes this Interceptor.

### Try it out locally:

1. To create the NamespacedInterceptor and all related resources, run:

   ```bash
   kubectl apply -f .
   ```

1. Port forward:

   ```bash
   kubectl port-forward service/el-example-interceptor-cel 8080 &
   ```

1. Test by sending the sample payload.

   ```bash
   curl -k -v     -H 'X-GitHub-Event: pull_request'     -H 'X-Hub-Signature: sha1=8d7c4d33686fd908394208a07d997b8f5bd70aa6'     -H 'Content-Type: application/json'     -d '{"head_commit":{"id":"28911bbb5a3e2ea034daf1f6be0a822d50e31e73"},"action": "opened", "pull_request":{"head":{"sha": "28911bbb5a3e2ea034daf1f6be0a822d50e31e73"}},"repository":{"clone_url": "https://github.com/tektoncd/triggers.git", "url":"https://github.com/tektoncd/triggers.git"}}'     http://localhost:8080
   ```

   The response status code should be `202 Accepted`


1. You should see a new TaskRun that got created:

   ```bash
   kubectl get taskruns | grep example-interceptor-run-
   ```
