curl -v \
-H 'X-GitHub-Event: pull_request' \
-H 'X-Hub-Signature-256: sha256=407de896d43ea3605e0e897112493c5f81eb052bff0b48783f7dfa264e3dceae' \
-H 'Content-Type: application/json' \
-d '{"action": "opened","number": 1503,"repository":{"full_name": "tektoncd/triggers", "clone_url": "https://github.com/tektoncd/triggers.git"}, "sender":{"login": "khrm"}}' \
http://localhost:8080
