# Tekton Triggers Official Release Cheat Sheet

This doc is a condensed version of our [tekton/README.md](./README.md) as
well as instructions from
[our plumbing repo](https://github.com/tektoncd/plumbing/tree/master/tekton/resources/release/README.md#create-draft-release).

These steps provide a no-frills guide to performing an official release
of Tekton Triggers. To follow these steps you'll need a checkout of
the triggers repo, a terminal window and a text editor.

1. `cd` to root of Triggers git checkout.

2. Point `kubectl` at dogfooding cluster.

    ```bash
    gcloud container clusters get-credentials dogfooding --zone us-central1-a --project tekton-releases
    ```

3. Edit `tekton/resources.yaml`. Add pipeline resource for new version.

    ```yaml
    apiVersion: tekton.dev/v1alpha1
    kind: PipelineResource
    metadata:
      name: tekton-triggers-git-$UPDATE_ME # UPDATE THIS. Example: tekton-triggers-git-v0-6-2
    spec:
      type: git
      params:
      - name: url
        value: https://github.com/tektoncd/triggers
      - name: revision
        value: # UPDATE THIS with the COMMIT SHA that you want to release. Example : 33e0847e67fc9804689e50371746c3cdad4b0a9d
    ```

4. `kubectl apply -f tekton/resources.yaml`

5. Create environment variables for bash scripts in later steps.

    ```bash
    VERSION_TAG=# UPDATE THIS. Example: v0.6.2
    PREVIOUS_VERSION_TAG=# UPDATE THIS. Example v0.6.0. Used to calculate release notes
    GIT_RESOURCE_NAME=# UPDATE THIS. Example: tekton-triggers-git-v0-6-2
    IMAGE_REGISTRY=gcr.io/tekton-releases
    ```

6. Confirm commit SHA matches what you want to release.

    ```bash
    kubectl get pipelineresource "$GIT_RESOURCE_NAME" -o=jsonpath="{'Target Revision: '}{.spec.params[?(@.name == 'revision')].value}{'\n'}"
    ```

7. Execute the release pipeline.

    **If you are backporting fixes to an older release, include this flag: `--param=releaseAsLatest="false"`**

    ```bash
      tkn pipeline start \
        --param=versionTag=${VERSION_TAG} \
        --serviceaccount=release-right-meow \
        --resource=source-repo=${GIT_RESOURCE_NAME} \
        --resource=bucket=tekton-triggers-bucket \
        --resource=builtEventListenerSinkImage=event-listener-sink-image \
        --resource=builtControllerImage=triggers-controller-image \
        --resource=builtWebhookImage=triggers-webhook-image \
        --resource=notification=post-release-trigger \
        triggers-release
    ```

8. Watch logs of triggers-release.

9. The YAMLs are now released! Anyone installing Tekton Triggers will now get the new version. Time to create a new GitHub release announcement:

    1. Run Draft Release Task.
      ```bash
      tkn task start \
        -i source=${GIT_RESOURCE_NAME} \
        -p package=tektoncd/triggers \
        -p release-tag=${VERSION_TAG} \
        -p previous-release-tag=${PREVIOUS_VERSION_TAG} \
        create-draft-triggers-release
      ```

    1. Watch logs of create-draft-release TaskRun for errors.

        ```bash
        tkn tr logs -f create-draft-release-run-# this will end with a random string of characters
        ```

    1. On successful completion, a URL will be logged. Visit that URL and sort the
    release notes. **Double-check that the list of commits here matches your expectations
    for the release.** You might need to remove incorrect commits or copy/paste commits
    from the release branch. Refer to previous releases to confirm the expected format.

    1. Publish the GitHub release once all notes are correct and in order.

10. Edit `README.md` on `master` branch, add entry to docs table with latest release links.

11. Push & make PR for updated `README.md`

12. **Important: Stop pointing `kubectl` at dogfooding cluster.**

    ```bash
    kubectl config use-context my-dev-cluster
    ```

13. Test release that you just made.

    ```bash
    # Test latest
    kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
    ```

    ```bash
    # Test backport
    kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/previous/v0.6.1/release.yaml
    ```

14. Announce the release in Slack channels #general and #triggers.

Congratulations, you're done!
