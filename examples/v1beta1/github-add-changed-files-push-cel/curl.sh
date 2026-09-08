# Re-generate the signature after changing the payload:
#   echo -n '<payload>' | openssl dgst -sha256 -hmac "$(kubectl get secret github-secret -o jsonpath='{.data.secretToken}' | base64 -d)"
curl -v \
-H 'X-GitHub-Event: push' \
-H 'X-Hub-Signature-256: sha256=8f6ceb5e0ed5f38f7361a63eb4da0c061be76f779b4b229a32f0f1f72f330273' \
-H 'Content-Type: application/json' \
-d '{"repository":{"full_name":"testowner/testrepo","clone_url":"https://github.com/testowner/testrepo.git"},"commits":[{"added":["api/v1beta1/tektonhelperconfig_types.go","config/crd/bases/tekton-helper..com_tektonhelperconfigs.yaml"],"removed":["config/samples/tektonhelperconfig-oomkillpipeline.yaml","config/samples/tektonhelperconfig-timeout.yaml"],"modified":["controllers/tektonhelperconfig_controller.go"]}]}' \
http://localhost:8080