## Example: Use Entire Request Body

This example demonstrates how a user can use `$(body)` in a [TriggerBinding](https://github.com/tektoncd/triggers/blob/master/docs/triggerbindings.md) to pass the entire request body as a single parameter. The example creates a TaskRun that prints the entire body of the HTTP request.

### Try it out locally

1. Create the ServiceAccount, Role, and RoleBinding:

   ```shell script
   kubectl apply -f examples/role-resources/triggerbinding-roles
   kubectl apply -f examples/role-resources/
   ```

1. Create the TriggerTemplate, TriggerBinding, and EventListener:

   ```shell script
   kubectl apply -f examples/use-entire-request-body/use-entire-request-body.yaml
   ```

1. Port forward:

   ```shell script
   kubectl port-forward \
    "$(kubectl get pod --selector=eventlistener=use-entire-request-body -oname)" \
     8080
   ```

   **Note**: Instead of port forwarding, you can set the
   [`serviceType`](https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#serviceType)
   to `LoadBalancer` to expose the EventListener with a public IP.

1. Test by sending an HTTP request to the EventListener:

   ```shell script
   curl -v -X POST http://localhost:8080 \
   -d '{
      "hello":
      {
         "this": "is"
      },
      "a":
      {
         "test": "."
      }
   }'
   ```

   The response status code should be `201 Created`

1. You should see a new TaskRun that got created:

   ```shell script
   kubectl get taskruns | grep use-entire-request-body
   ```

1. View the TaskRun logs to see that the request body is printed in the TaskRun:

   The end of the TaskRun logs will print the following:

   ```
   {
     "a": {
       "test": "."
     },
     "hello": {
       "this": "is"
     }
   }
   ```
