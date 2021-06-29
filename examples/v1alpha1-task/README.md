## v1alpha1 Task EventListener

Creates an EventListener that creates a v1alpha1 TaskRun.

### Try it out locally:

1. Create the v1alpha1 EventListener and its ServiceAccount:

   ```shell script
   kubectl apply -f examples/v1alpha1-task/
   ```

1. Port forward:

   ```shell script
   kubectl port-forward service/el-v1alpha1-task-listener 8080
   ```

1. Test by sending the sample payload.

   ```shell script
   curl -v \
   -H 'Content-Type: application/json' \
   --data "{}" \
   http://localhost:8080
   ```

   The response status code should be `202 Accepted`

1. You should see a new TaskRun that got created:

   ```shell script
   kubectl get taskruns | grep v1alpha1-task-run-
   ```
