curl -v \
-H 'X-GitHub-Event: pull_request' \
-H 'X-Hub-Signature: sha1=ba0cdc263b3492a74b601d240c27efe81c4720cb' \
-H 'Content-Type: application/json' \
-d '{"action": "opened", "pull_request":{"head":{"sha": "28911bbb5a3e2ea034daf1f6be0a822d50e31e73"}},"repository":{"clone_url": "https://github.com/tektoncd/triggers.git"}}' \
http://localhost:8080