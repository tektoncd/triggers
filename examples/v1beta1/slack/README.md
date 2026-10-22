## Slack slash command  EventListener

Creates an EventListener that listens for Slack  events.

### Try it out locally:

1. Replace the value in `secret.yaml` with your Slack app signing secret, then create the slack trigger and all related resources:

   ```bash
   kubectl apply -f .
   ```

1. Port forward:

   ```bash
   kubectl port-forward service/el-slack-listener 8080
   ```

1. Test by sending the sample payload.

   ```bash
   bash curl.sh
   ```

   `curl.sh` reads the signing secret back out of the `slack-secret` Secret and
   generates a fresh `X-Slack-Request-Timestamp` and `X-Slack-Signature` for each
   run. Slack includes the timestamp in the signed string, so the signature cannot
   be hardcoded — the interceptor rejects any request whose timestamp is more than
   five minutes old.

   The response status code should be `202 Accepted`


1. You should see a new TaskRun that got created:

   ```bash
   kubectl get taskruns | grep slack-run-
   ```
