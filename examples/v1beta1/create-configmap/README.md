## Create Configmap with EventListener

Creates an EventListener that will create a configmap as an example how to create non tekton resources with Triggers.

### Try it out locally:

1. To create the Custom trigger and all related resources, run:

   ```bash
   kubectl apply -f .
   ```

1. Port forward:

   ```bash
   kubectl port-forward service/el-create-configmap-listener 8080
   ```

1. Test by sending the sample payload.

   ```bash
   curl -v \
   -H 'Content-Type: application/json' \
   -d '{"action": "opened"}' \
   http://localhost:8080
   ```

   The response status code should be `202 Accepted`

1. You should see a new ConfigMap that got created:

   ```bash
   kubectl get configmaps | grep sample-
   ```
