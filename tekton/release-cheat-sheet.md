# Tekton Triggers Official Release Cheat Sheet

These steps provide a no-frills guide to performing an official release
of Tekton Triggers. To follow these steps you'll need a checkout of
the triggers repo, a terminal window and a text editor.

1. [Setup a context to connect to the dogfooding cluster](#setup-dogfooding-context) if you haven't already.

1. `cd` to root of Triggers git checkout.

1. Make sure the release `Task` and `Pipeline` are up-to-date on the
   cluster.

   - [publish-triggers-release](https://github.com/tektoncd/triggers/blob/main/tekton/publish.yaml)

     This task uses [ko](https://github.com/google/ko) to build all container images we release and generate the `release.yaml`
     ```shell script
     kubectl apply -f tekton/publish.yaml
     ```
   - [triggers-release](https://github.com/tektoncd/triggers/blob/main/tekton/release-pipeline.yaml)
     ```shell script
     kubectl apply -f tekton/release-pipeline.yaml
     ```

1. Select the commit you would like to build the release from, most likely the
   most recent commit at https://github.com/tektoncd/triggers/commits/main
   and note the commit's hash.

1. Create environment variables for bash scripts in later steps.

    ```bash
    VERSION_TAG=# UPDATE THIS. Example: v0.6.2
    TRIGGERS_RELEASE_GIT_SHA=# SHA of the release to be released
    ```

1. Confirm commit SHA matches what you want to release.

    ```bash
    git show $TRIGGERS_RELEASE_GIT_SHA
    ```

1. Create a workspace template file:

   ```bash
   cat <<EOF > workspace-template.yaml
   spec:
     accessModes:
     - ReadWriteOnce
     resources:
       requests:
         storage: 1Gi
   EOF
   ```

1. Execute the release pipeline.

    **If you are back-porting include this flag: `--param=releaseAsLatest="false"`**

    ```bash
    tkn --context dogfooding pipeline start triggers-release \
      --param=gitRevision="${TRIGGERS_RELEASE_GIT_SHA}" \
      --param=versionTag="${VERSION_TAG}" \
      --param=serviceAccountPath=release.json \
      --param=releaseBucket=gs://tekton-releases/triggers \
      --workspace name=release-secret,secret=release-secret \
      --workspace name=workarea,volumeClaimTemplateFile=workspace-template.yaml
    ```

1. Watch logs of triggers-release.

1. Once the pipeline run is complete, check its results:

   ```bash
   tkn --context dogfooding pr describe <pipeline-run-name>

   (...)
   📝 Results

   NAME                    VALUE
   commit-sha                 6ea31d92a97420d4b7af94745c45b02447ceaa19
   release-file               https://storage.googleapis.com/tekton-releases/triggers/previous/v0.13.0/release.yaml
   release-file-no-tag        https://storage.googleapis.com/tekton-releases/triggers/previous/v0.13.0/release.notag.yaml
   interceptors-file          https://storage.googleapis.com/tekton-releases/triggers/previous/v0.13.0/interceptors.yaml
   interceptors-file-no-tag   https://storage.googleapis.com/tekton-releases/triggers/previous/v0.13.0/interceptors.notag.yaml

   (...)
   ```

   The `commit-sha` should match `$TRIGGERS_RELEASE_GIT_SHA`.
   The two URLs can be opened in the browser or via `curl` to download the release manifests.

    1. The YAMLs are now released! Anyone installing Tekton Triggers will now get the new version. Time to create a new GitHub release announcement:

    1. Create additional environment variables

    ```bash
    TRIGGERS_OLD_VERSION=# Example: v0.11.1
    TEKTON_PACKAGE=tektoncd/triggers
    ```

    1. Find the Rekor UUID for the release

    ```bash
    RELEASE_FILE=https://storage.googleapis.com/tekton-releases/triggers/previous/${VERSION_TAG}/release.yaml
    CONTROLLER_IMAGE_SHA=$(curl $RELEASE_FILE | sed -n 's/"//g;s/.*gcr\.io.*controller.*@//p;')
    REKOR_UUID=$(rekor-cli search --sha $CONTROLLER_IMAGE_SHA | grep -v Found | head -1)
    echo -e "CONTROLLER_IMAGE_SHA: ${CONTROLLER_IMAGE_SHA}\nREKOR_UUID: ${REKOR_UUID}"
    ```

    1. Execute the Draft Release task.

    ```bash
    tkn --context dogfooding pipeline start \
        --workspace name=shared,volumeClaimTemplateFile=workspace-template.yaml \
        --workspace name=credentials,secret=release-secret \
        -p package="${TEKTON_PACKAGE}" \
        -p git-revision="${TRIGGERS_RELEASE_GIT_SHA}" \
        -p release-tag="${VERSION_TAG}" \
        -p previous-release-tag="${TRIGGERS_OLD_VERSION}" \
        -p release-name="Tekton Triggers" \
        -p bucket="gs://tekton-releases/triggers" \
        -p rekor-uuid="$REKOR_UUID" \
        release-draft
    ```

    1. Watch logs of create-draft-release

    1. On successful completion, a URL will be logged. Visit that URL and look through the release notes.
      1. Manually add upgrade and deprecation notices based on the generated release notes
      1. Double-check that the list of commits here matches your expectations
         for the release. You might need to remove incorrect commits or copy/paste commits
         from the release branch. Refer to previous releases to confirm the expected format.
      1. In the section **Installation one-liner**, add the install instruction for interceptors also. 
         ```bash
            kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/previous/${VERSION_TAG}/interceptors.yaml
         ```
      1. In the section **Attestation**, modify it for inteceptors.yaml also.
         ```bash
         RELEASE_FILE=https://storage.googleapis.com/tekton-releases/triggers/previous/${VERSION_TAG}/release.yaml
         INTERCEPTORS_FILE=https://storage.googleapis.com/tekton-releases/triggers/previous/${VERSION_TAG}/interceptors.yaml
         REKOR_UUID=$REKOR_UUID

         # Obtains the list of images with sha from the attestation
         REKOR_ATTESTATION_IMAGES=$(rekor-cli get --uuid "$REKOR_UUID" --format json | jq -r .Attestation | jq -r '.subject[]|.name + ":v0.23.0@sha256:" + .digest.sha256')


         # Download the release file
         curl "$RELEASE_FILE" > release.yaml
         curl "$INTERCEPTORS_FILE" >> release.yaml
          ```
    1. Un-check the "This is a pre-release" checkbox since you're making a legit for-reals release!

    1. Publish the GitHub release once all notes are correct and in order.

1. Edit `releases.md` on the `main` branch, add an entry for the release.
   - In case of a patch release, replace the latest release with the new one,
     including links to docs and examples. Append the new release to the list
     of patch releases as well.
   - In case of a minor or major release, add a new entry for the
     release, including links to docs and example
   - Check if any release is EOL, if so move it to the "End of Life Releases"
     section

1. Push & make PR for updated `releases.md`

1. Test release that you just made against your own cluster (note `--context my-dev-cluster`):

    ```bash
    # Test latest
    kubectl --context my-dev-cluster apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
    kubectl --context my-dev-cluster apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml
    ```

    ```bash
    # Test backport
    kubectl --context my-dev-cluster apply --filename https://storage.googleapis.com/tekton-releases/triggers/previous/v0.12.1/release.yaml
    # NOTE: Some older releases might not have a separate interceptors.yaml as they used to be bundled in release.yaml
    kubectl --context my-dev-cluster apply --filename https://storage.googleapis.com/tekton-releases/triggers/previous/v0.12.1/interceptors.yaml
    ```

1. For major releases, the [website sync configuration](https://github.com/tektoncd/website/blob/main/sync/config/triggers.yaml)
   to include the new release.

1. Announce the release in Slack channels #general, #triggers and #announcements.

Congratulations, you're done!

## Setup dogfooding context

1. Configure `kubectl` to connect to
   [the dogfooding cluster](https://github.com/tektoncd/plumbing/blob/main/docs/dogfooding.md):

    ```bash
    gcloud container clusters get-credentials dogfooding --zone us-central1-a --project tekton-releases
    ```

1. Give [the context](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/)
   a short memorable name such as `dogfooding`:

   ```bash
   kubectl config rename-context gke_tekton-releases_us-central1-a_dogfooding dogfooding
   ```

## Important: Switch `kubectl` back to your own cluster by default.

```bash
kubectl config use-context my-dev-cluster
```
