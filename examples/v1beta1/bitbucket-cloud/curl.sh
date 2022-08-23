curl -v \
-H 'X-Event-Key: repo:push' \
-d '{"push": {"changes":[{"old":{}, "new":{"name":"bb-cloud-test-1"}}]},"repository":{"links": {"html":{"href": "https://bitbucket.org/savitaredhat/bitbuckettest-customer"}},"name":"bitbuckettest-customer"}, "actor":{"display_name": "savita"}}' \
http://localhost:8080