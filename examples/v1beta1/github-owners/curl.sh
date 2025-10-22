curl -v \
-H 'X-GitHub-Event: pull_request' \
-H 'X-Hub-Signature-256: sha256=a7b3a3840860ef271afde557e8b6c89cc69539a396417f93d847e1890d3c8184' \
-H 'Content-Type: application/json' \
-d '{"action": "opened","number": 1503,"repository":{"full_name": "tektoncd/triggers", "clone_url": "https://github.com/tektoncd/triggers.git"}, "sender":{"login": "dibyom"}}' \
http://localhost:8080