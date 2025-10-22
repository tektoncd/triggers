curl -X POST \
  http://localhost:8080 \
  -H 'Content-Type: application/json' \
  -H 'X-Hub-Signature-256: sha256=5bfc4c007697a264f4255882e9fbffe34cbe1cf6040118db7e017e6200f45acd' \
  -d '{
	"repository":
	{
		"url": "https://github.com/tektoncd/triggers.git"
	}
}'