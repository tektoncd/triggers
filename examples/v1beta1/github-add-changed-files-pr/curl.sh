curl -v \
-H 'X-GitHub-Event: pull_request' \
-H 'Content-Type: application/json' \
-d '{"action": "opened","number": 1503,"pull_request": {"head": {"sha": "16dd484bb4888dd30154f5ccb765beae1aaf72de"}},"repository": {"full_name": "tektoncd/triggers","clone_url": "https://github.com/tektoncd/triggers.git"}}' \
http://localhost:8080