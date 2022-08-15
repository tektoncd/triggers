## AzureRepo pull request EventListener

Creates an EventListener that listens for Azure Repo pull request.

### Pre-requisites

1. Should have access to azure repo
1. Should have URL accessible publicly to configure in webhook

### Steps to try:

1. To create the AzureRepo pull request eventlistener and all related resources, run:

   ```bash
   kubectl apply -f .
   ```

1. To get the eventlistener URL, run:

   ```bash
   kubectl get el
   ```

1. Login to AzureRepo and perform steps to configure event type to pull request created and eventlistener URL in webhook
1. Send pull request to AzureRepo

1. You should see a new TaskRun that got created:

   ```bash
   kubectl get taskruns | grep azurerepo-pr-listener-run-
   ```
