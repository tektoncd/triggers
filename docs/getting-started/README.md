# Tutorial: Getting started with Tekton Triggers

The following tutorial walks you through building and deploying a Docker image using 
Tekton Triggers to detect a GitHub webhook request and execute a `Pipeline`.

## Overview

In this tutorial, you will:

1. Set up a [`Pipeline`](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md) that builds a Docker image using
   [kaniko](https://github.com/GoogleContainerTools/kaniko) and deploys it locally on your Kubernetes cluster. The workflow
   in the `Pipeline` is as follows:
   1. Retrieve the source code.
   1. Build and push the source code into a Docker image.
   1. Push the image to the specified repository.
   1. Run the image locally.

2. Set up an [`EventListener`](https://github.com/tektoncd/triggers/blob/main/docs/eventlisteners.md) that accepts and processes GitHub push events.

3. Set up a [`TriggerTemplate`](https://github.com/tektoncd/triggers/blob/main/docs/triggertemplates.md) that instantiates a
   [`PipelineResource`](https://github.com/tektoncd/pipeline/blob/main/docs/resources.md) and executes a [`PipelineRun`](https://github.com/tektoncd/pipeline/blob/main/docs/pipelineruns.md)
   and its associated ['TaskRuns'](https://github.com/tektoncd/pipeline/blob/main/docs/taskruns.md) when the `EventListener` detects the push event from a GitHub repository.

4. Run the completed stack to experience Tekton Triggers in action.

## Prerequisites

Before you begin, you must satisfy the following prerequisites:

- [Set up a Kubernetes cluster](https://kubernetes.io/docs/setup/) that you can publicly access over the Internet.
- [Install Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md#installing-tekton-pipelines).
  Tekton Triggers installs on top of Tekton Pipelines.
- [Install Tekton Triggers](../install.md).
- Have a GitHub repository and select a Dockerfile within that repository as your build object.
  For this tutorial, you can fork our [example repo](https://github.com/iancoffey/ulmaceae).
  You must clone the selected repository locally.

## Configure your cluster

Now that you have your Kubernetes cluster up and running, you must set up your namespace and RBAC.
You will keep all of the artifacts for this tutorial within this namespace. This way, you can easily
start over by deleting and recreating this namespace if necessary. 

**Note:** Record your `ingress` sub-domain or the external IP address of your
cluster as you will need it to create your GitHub webhook later in this tutorial.

Configure your cluster as follows:

1. Create a namespace named `getting-started` using the following command:

   ```
   kubectl create namespace getting-started
   ```

2. Create the [`admin` user, role, and rolebinding](./rbac/admin-role.yaml) using the following command:
   
   ```
   kubectl -n getting-started apply -f ./docs/getting-started/rbac/admin-role.yaml \
               -f ./docs/getting-started/rbac/clusterrolebinding.yaml
   ```
3. (Optional) If you have already provisioned a cluster secret for a "Let's Encrypt" certificate,
   you must export it and then import it into your `getting-started` namespace. For example:

   ```bash
	kubectl get secret <name> --namespace=<namespace> -o yaml |\
	   grep -v '^\s*namespace:\s' |\
	   kubectl apply --namespace=<new namespace> -f -
   ```
4. Create the [`create-webhook` user, role, and rolebinding](./rbac/webhook-role.yaml) using the following command:

   ```
   kubectl -n getting-started apply -f ./docs/getting-started/rbac/webhook-role.yaml
   ```
  This allows your webhook to work with Tekton Triggers.

5. (Optional) If your cluster doesn't have access to your Docker registry, you must add a secret to both your cluster
   and the `pipeline.yaml` file in this tutorial as follows:
   1. Add a secret to your cluster as described in [Configuring `Task` execution credentials](https://github.com/tektoncd/pipeline/blob/main/docs/tutorial.md#configuring-task-execution-credentials).
   2. Add the secret you created in the previous step to your `pipeline.yaml` file by adding the following to each `Task` within the file:

   ```
     env:
       - name: "DOCKER_CONFIG"
         value: "/tekton/home/.docker/"
   ```

## Install the example resources

You are now ready to install the example resources to use in the tutorial:

 - A `Pipeline`
 - A `TriggerTemplate`
 - A `TriggerBinding`
 - An `EventListener`

1. Install the example [`Pipeline`](./pipeline.yaml) using the following command:

   ```
   kubectl -n getting-started apply -f ./docs/getting-started/pipeline.yaml
   ```

2. Install the example [Triggers resources](./triggers.yaml) as follows:
   1. Update the `triggers.yaml` file with the repository to which you want your `Pipeline` to push
      the Docker image binary by replacing the `DOCKERREPO-REPLACEME` placeholder string throughout
      the file.
   2. Apply the updated `triggers.yaml` file on your cluster using the following command:
   ```
   kubectl -n getting-started apply -f ./docs/getting-started/triggers.yaml
   ```

Your Tekton stack is now configured to detect and respond to GitHub events.

## Create and execute the ingress and webhook `Tasks`

Now, you must create and execute the following `Tasks`:
- Ingress `Task` - exposes the `EventListener` at a publicly accessible address to which
  the GitHub webhook can send events.
- Webhook `Task` - creates the Github webhook that sends events to your `EventListener`.

1. Create the ingress `Task`:

   ```
   kubectl -n getting-started apply -f ./docs/getting-started/create-ingress.yaml
   ```

2. Create the webhook `Task`: 

   ```
   kubectl -n getting-started apply -f ./docs/getting-started/create-webhook.yaml
   ```

3. Update the `TaskRun` for the ingress `Task`. At the minimum, you must update the `ExternalDomain`
   field in the `docs/getting-started/ingress-run.yaml` file to match your DNS name. You might also
   need to modify other settings as appropriate.

4. Run the ingress `Task`:

   ```
   kubectl -n getting-started apply -f docs/getting-started/ingress-run.yaml
   ```

5. Create a [GitHub Personal Access Token](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line#creating-a-token)
   with the following access privileges:
   - `public_repo`
   - `admin:repo_hook`

   This token can contain any plain text string.

6. Add the token to the `docs/getting-started/secret.yaml` file. Do NOT `base64`-encode the token when adding it to the `secret.yaml` file.

7. Create the required secret with the following command:
   ```
   kubectl -n getting-started apply -f docs/getting-started/secret.yaml
   ```

8. Update the `TaskRun` for the webhook `Task`. At the minimum, you must update the following fields
   in the `docs/getting-started/webhook-run.yaml` file:
   - `GitHubOrg` - the GitHub organization you're using for the namespace in this tutorial.
   - `GitHubUser` - your GitHub username.
   - `GitHubRepo` - the GitHub repository you're using for this tutorial.
   - `ExternalDomain` - set this to a value appropriate to your environment: the external domain of the event listener instance.
   - `GitHubDomain` (optional) - if you are using github enterprise, set this to your GitHub domain (e.g. `git.corp.com`)

9. Run the webhook `Task`:
   ``` 
   kubectl -n getting-started apply -f docs/getting-started/webhook-run.yaml
   ```

## Run the completed Tekton Triggers stack

You are now ready to experience Tekton Triggers in action! Do the following:

1. Make an empty commit and push it to your repository:
   ```
   git commit -a -m "build commit" --allow-empty && git push origin mybranch
   ```

2. Monitor the execution of your `Tasks`:
   - Monitor the image builder `Task` using the following command:
     ```
     kubectl logs -l somelabel=somekey --all-containers
     ```
   - Monitor the deployer `Task` using the following command:
     ```
     kubectl -n getting-started logs -l tekton.dev/pipeline=getting-started-pipeline --all-containers
     ```
     
   You can see that the system is working and that pushing images to your repository results in
   a running `Pod` using the following command:
   ```	
   kubectl -n getting-started logs tekton-triggers-built-me --all-containers
   ```

Congratulations! Your new image has been retrieved, tested, vetted, built, docker-pushed and pulled,
and is now running on your cluster as a `Pod`.

## Cleaning up

To clean up, simply delete the `getting-started` namespace using the following command:
```
kubectl delete namespace getting-started
```
