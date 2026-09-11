# Re-generate the signature after changing the payload:
#   echo -n '<payload>' | openssl dgst -sha256 -hmac "$(kubectl get secret github-secret -o jsonpath='{.data.secretToken}' | base64 -d)"
curl -v \
-H 'X-GitHub-Event: pull_request' \
-H 'X-Hub-Signature-256: sha256=687613a04e426350095daa3b8fd9d1a4615ed770816cd9a68adfcb47a602adda' \
-H 'Content-Type: application/json' \
-d '{"action": "opened","number": 1503,"pull_request": {"head": {"sha": "16dd484bb4888dd30154f5ccb765beae1aaf72de"}},"repository": {"full_name": "tektoncd/triggers","clone_url": "https://github.com/tektoncd/triggers.git"}}' \
http://localhost:8080