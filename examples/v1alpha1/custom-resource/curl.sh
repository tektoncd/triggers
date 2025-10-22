curl -v \
-H 'X-GitHub-Event: pull_request' \
-H 'X-Hub-Signature-256: sha256=c26dd919ebe335852219c49f74c4b24f1c62c93c77294be3ac6d8f2e4691a023' \
-H 'Content-Type: application/json' \
-d '{"action": "opened", "pull_request":{"head":{"sha": "28911bbb5a3e2ea034daf1f6be0a822d50e31e73"}},"repository":{"clone_url": "https://github.com/tektoncd/triggers.git"}}' \
http://127.0.0.1:$1 -H "Host: $2"
