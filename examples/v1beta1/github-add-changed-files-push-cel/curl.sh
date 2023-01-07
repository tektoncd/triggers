curl -v \
-H 'X-GitHub-Event: push' \
-H 'Content-Type: application/json' \
-d '{"repository":{"full_name":"testowner/testrepo","clone_url":"https://github.ford.com/testowner/testrepo.git"},"commits":[{"added":["api/v1beta1/tektonhelperconfig_types.go","config/crd/bases/tekton-helper.ford.com_tektonhelperconfigs.yaml"],"removed":["config/samples/tektonhelperconfig-oomkillpipeline.yaml","config/samples/tektonhelperconfig-timeout.yaml"],"modified":["controllers/tektonhelperconfig_controller.go"]}]}' \
http://localhost:8080