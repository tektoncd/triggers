<!--
---
title: "Install and set up Tekton Triggers"
linkTitle: "Install and set up Tekton Triggers"
weight: 2
---
-->

This document shows you how to install and set up Tekton Triggers.

## Prerequisites

-   [Kubernetes] cluster version 1.18 or later.
-   [Kubectl].
-   [Tekton Pipelines][pipelines-install].
-   Grant `cluster-admin` privileges to the user that installed Tekton Pipelines. See
    the [kubernetes role-based access control (RBAC) docs][rbac].

## Installation

1.  Log on to your Kubernetes cluster with the same user account that installed
    Tekton Pipelines.

1.  Depending on which version of Tekton Triggers you want to install, run one
    of the following commands:

    -   **Latest official release**

        ```bash
        kubectl apply --filename \
        https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
        kubectl apply --filename \
        https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml
        ```

    -   **Nightly release**

        ```bash
        kubectl apply --filename \
        https://storage.googleapis.com/tekton-releases-nightly/triggers/latest/release.yaml
        kubectl apply --filename \
        https://storage.googleapis.com/tekton-releases-nightly/triggers/latest/interceptors.yaml
        ```

    -   **Specific Release**

        ```bash
        kubectl apply --filename \
        https://storage.googleapis.com/tekton-releases/triggers/previous/VERSION_NUMBER/release.yaml
        kubectl apply --filename \
        https://storage.googleapis.com/tekton-releases/triggers/previous/VERSION_NUMBER/interceptors.yaml
        ```

        Replace `VERSION_NUMBER` with the numbered version you want to install.
        For example, `v0.19.1`.

    -   **Untagged Release**

        If your container runtime does not support `image-reference:tag@digest` (for
        example, `cri-o` used in OpenShift 4.x):

        ```bash
        kubectl apply --filename \
        https://storage.googleapis.com/tekton-releases/triggers/latest/release.notags.yaml
        kubectl apply --filename \
        https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.notags.yaml
        ```

1.  To monitor the installation, run:

    ```bash
    kubectl get pods --namespace tekton-pipelines --watch
    ```

    When all components show `1/1` under the `READY` column, the installation is
    complete. Hit *Ctrl + C* to stop monitoring.

## Customization options

You can customize the behavior of the Triggers Controller changing some values
in the `config/feature-flags-triggers.yaml` file.

+ Enable alpha features. Set the value of `enable-api-fields:` to `"alpha"`, the
  default value is `"stable"`. This flag only applies to the v1beta1 API
  version.

+ Exclude labels. Set the `labels-exclusion-pattern:` field to a regex  pattern.
  Labels that match this pattern are excluded from getting added to the
  resources created by the EventListener. By default this field is empty, so all
  labels added to the EventListener are propagated down.

## Further reading

+ [Get started with Tekton Triggers][get-started]
+ [Explore Tekton Triggers code examples][code-examples]

[kubernetes]: https://kubernetes.io/docs/concepts/overview/what-is-kubernetes/
[kubectl]: https://kubernetes.io/docs/tasks/tools/#kubectl
[triggers]: https://tekton.dev/docs/triggers/
[get-started]: https://github.com/tektoncd/triggers/blob/main/docs/getting-started
[code-examples]: https://github.com/tektoncd/triggers/tree/main/examples
[pipelines-install]: https://github.com/tektoncd/pipeline/blob/main/docs/install.md
[rbac]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
