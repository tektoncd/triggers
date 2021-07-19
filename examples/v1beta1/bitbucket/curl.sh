curl -v \
-H 'X-Event-Key: repo:refs_changed' \
-H 'X-Hub-Signature: sha1=b3fdaf5d1a47e57527764a233659c650a11abdd8' \
-d '{"repository": {"links": {"clone": [{"href": "http://localhost:7990/scm/~test/helloworld.git", "name": "http"}, {"href": "ssh://git@localhost:7999/~test/helloworld.git", "name": "ssh"}]}}, "changes": [{"ref": {"displayId": "main"}}]}' \
http://localhost:8080