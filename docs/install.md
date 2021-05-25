<!--
---
linkTitle: "Setting Up Tekton Triggers"
weight: 2
---
-->
# Setting Up Tekton Triggers

Follow the steps below to set up an official release of Tekton Triggers on your Kubernetes cluster.
If you want to test Triggers from `HEAD`, see the
[Tekton Developer Guide](https://github.com/tektoncd/triggers/blob/main/DEVELOPMENT.md#install-triggers).

## Prerequisites

You must satisfy the following prerequisties to set up Tekton Triggers:

* You must have a Kubernetes Cluster running Kubernetes 1.18 or above.

  You can use [`kind`](https://kind.sigs.k8s.io/) to quickly create a local cluster with RBAC enabled for testing purposes:

  * Install `kind` as described in [Installation](https://kind.sigs.k8s.io/docs/user/quick-start/#installation).

  * Create a cluster as described in [Creating a Cluster](https://kind.sigs.k8s.io/docs/user/quick-start/#creating-a-cluster).

* You must have Tekton Pipelines installed on your Kubernetes cluster.

  For instructions, see [Installing Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md).

* You must grant the `cluster-admin` privilege to the user with which you installed Tekton Pipelines.

  For instructions, see [Role-based access control](https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control#prerequisites_for_using_role-based_access_control).

## Installing Tekton Triggers on Your Cluster

1. Log on to your Kubernetes cluster as the user with which you installed Tekton Pipelines.

1.  Use the [`kubectl apply`](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#apply) command to install the latest release of Tekton Triggers and its dependencies:

    ```bash
    kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
    kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml
    ```

    To install a specific release of Tekton Triggers, replace `latest` with the desired version number as shown in the following example:

    ```bash
    kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/previous/v0.1.0/release.yaml
    ```

    To install a nightly release, use the following command:

    ```bash
    kubectl apply --filename https://storage.googleapis.com/tekton-releases-nightly/triggers/latest/release.yaml
    ```

1.  Monitor the installation using the [`kubectl get`](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#get)
    command:

    ```bash
    kubectl get pods --namespace tekton-pipelines --watch
    ```

1. When all Tekton Triggers components report a status of `Running` press CTRL+C to stop monitoring.

You are now ready to configure Tekton Triggers for your workflow. For instructions, see the following:

- [Tekton Triggers Getting Started Guide](./getting-started/)
- [Tekton Triggers code examples](https://github.com/tektoncd/triggers/tree/main/examples)

## Customizing the Triggers Controller behavior

To customize the behavior of the Triggers Controller, modify the ConfigMap `feature-flags` as follows:

- `enable-api-fields`: set this flag to "stable" to allow only the
most stable features to be used. Set it to "alpha" to allow alpha
features to be used.

For example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: feature-flags
data:
  enable-api-fields: "alpha" # Allow alpha fields to be used in Tasks and Pipelines.
```