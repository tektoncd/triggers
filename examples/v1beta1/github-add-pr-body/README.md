## GitHub EventListener

Creates an EventListener that listens for GitHub webhook events and adds the
body of the associated pull request to the GitHub payload. This is useful for
running CI based on updates to comments on a pull request, since a plain
`issue_comment` event does not include the pull request body. The pull
request body is added to the `pr_body` property of the event payload in the
top-level `extensions` field.

The feature applies to `pull_request` events, and to `issue_comment` events
raised on a pull request (GitHub fires `issue_comment` both for comments on
plain issues and for comments on pull requests; when the comment is on a
plain issue there is no pull request body to add, and the interceptor is a
no-op).

### Try it out locally:

1. To create the GitHub trigger and all related resources, run:

   ```bash
   kubectl apply -f .
   ```

1. Port forward:

   ```bash
   kubectl port-forward service/el-github-add-pr-body-listener 8080
   ```

1. Test by sending the sample payload.

   ```bash
    curl -v \
    -H 'X-GitHub-Event: pull_request' \
    -H 'Content-Type: application/json' \
    -d '{"action": "opened","number": 1503,"pull_request": {"head": {"sha": "16dd484bb4888dd30154f5ccb765beae1aaf72de"}},"repository": {"full_name": "tektoncd/triggers","clone_url": "https://github.com/tektoncd/triggers.git"}}' \
    http://localhost:8080
   ```

   The response status code should be `202 Accepted`

   [`HMAC`](https://www.freeformatter.com/hmac-generator.html) tool used to create X-Hub-Signature-256.

   In [`HMAC`](https://www.freeformatter.com/hmac-generator.html) `string` is the *body payload ex:* `{"action": "opened", "pull_request":{"head":{"sha": "28911bbb5a3e2ea034daf1f6be0a822d50e31e73"}},"repository":{"clone_url": "https://github.com/tektoncd/triggers.git"}}`
   and `secretKey` is the *given secretToken ex:* `1234567`.

1. You should see a new TaskRun that got created:

   ```bash
   kubectl get taskruns | github-add-pr-body-run-
   ```

1. Get the pod created from the TaskRun and show the logs to see the pull request body:

   ```bash
   kubectl get pods | grep github-add-pr-body-run-
   kubectl logs <POD NAME>
   ```

### Private repositories

Fetching the pull request body requires calling the GitHub API. For public
repositories this works without authentication, subject to GitHub's low rate
limit for unauthenticated requests. For private repositories, or to avoid
that rate limit, supply a [GitHub personal access
token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
via the `personalAccessToken` field, the same way it is used for the
[`addChangedFiles`](../github-add-changed-files-pr) feature:

```yaml
- name: "addPRBody"
  value:
    enabled: true
    personalAccessToken:
      secretName: github-pat
      secretKey: token
```
