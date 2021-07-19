curl -v \
-H 'X-GitLab-Token: 1234567' \
-H 'X-Gitlab-Event: Push Hook' \
-H 'Content-Type: application/json' \
--data-binary "@examples/v1beta1/gitlab/gitlab-push-event.json" \
http://localhost:8080