curl -v \
-H 'X-GitHub-Event: pull_request' \
-H 'X-Hub-Signature-256: sha256=4564812b3882ca1531a3ecc4f481ee98a52a434141456455f25d92839d0d9572' \
-H 'Content-Type: application/json' \
-d '{"action":"opened","number":1503,"pull_request":{"url":"https://api.github.com/repos/tektoncd/triggers/pulls/1503","id":1,"number":1503,"state":"open","title":"Test PR","user":{"login":"dibyom"}},"repository":{"full_name":"tektoncd/triggers","clone_url":"https://github.com/tektoncd/triggers.git"},"sender":{"login":"dibyom"}}' \
http://localhost:8080