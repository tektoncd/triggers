## Slack slash command  EventListener

Creates an EventListener that listens for Slack  events.

### Try it out locally:

1. To create the slack trigger and all related resources, run:

   ```bash
   kubectl apply -f .
   ```

1. Port forward:

   ```bash
   kubectl port-forward service/el-slack-listener 8080
   ```

1. Test by sending the sample payload.

   ```bash
curl -v \
-H 'X-Slack-Signature: sha1=ba0cdc263b3492a74b601d240c27efe81c4720cb' \
-H 'Content-Type: application/x-www-form-urlencoded' \
-d 'token=EidhofDor5uIpqQ9RrtOVdnC&team_id=T04PK47eDS4&team_domain=demoworkspace-tid8978&channel_id=C04NETw4NBH&channel_name=sample-app&user_id=U04NVDwF7R8&&command=%2Fbuild&text=main+2222&api_app_id=A04NXU23L&is_enterprise_install=false&response_url=https%3A%2F%2Fhooks.slack.com%2Fcommands%2FT04PK47EDS4%2F4863712501879%2FdOMNffCDfTjlSskBrmB1bOtR&trigger_id=4890883491553.4801143489888.910b8eaae200b381834de25310583f74' \
http://localhost:8080
   ```

   The response status code should be `202 Accepted`


1. You should see a new TaskRun that got created:

   ```bash
   kubectl get taskruns | grep slack-run-
   ```
