curl -v \
-H 'X-GitHub-Event: pull_request' \
-H 'X-Hub-Signature: sha1=70d0ebf86a7973374898b7711acc0897616e2c93' \
-H 'Content-Type: application/json' \
-d '{"action": "opened","number": 1503,"repository":{"full_name": "tektoncd/triggers", "clone_url": "https://github.com/tektoncd/triggers.git"}, "sender":{"login": "dibyom"}}' \
http://localhost:8080