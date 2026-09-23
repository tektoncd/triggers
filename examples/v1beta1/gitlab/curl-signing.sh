#!/usr/bin/env bash
# Send a sample GitLab push event with Standard Webhooks signing headers.
# Matches examples/v1beta1/gitlab/secret.yaml signingToken.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BODY_FILE="${ROOT}/gitlab-push-event.json"
SIGNING_TOKEN='whsec_AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE='
WEBHOOK_ID='msg_local_test_1'
WEBHOOK_TIMESTAMP="$(date +%s)"

WEBHOOK_SIGNATURE="$(python3 - "${SIGNING_TOKEN}" "${WEBHOOK_ID}" "${WEBHOOK_TIMESTAMP}" "${BODY_FILE}" <<'PY'
import base64, hashlib, hmac, sys

token, msg_id, ts, path = sys.argv[1:5]
raw_key = base64.b64decode(token.removeprefix("whsec_"))
body = open(path, "rb").read()
digest = hmac.new(raw_key, f"{msg_id}.{ts}.".encode() + body, hashlib.sha256).digest()
print("v1," + base64.b64encode(digest).decode())
PY
)"

curl -v \
  -H 'X-GitLab-Token: 1234567' \
  -H 'X-Gitlab-Event: Push Hook' \
  -H 'Content-Type: application/json' \
  -H "Webhook-Id: ${WEBHOOK_ID}" \
  -H "Webhook-Timestamp: ${WEBHOOK_TIMESTAMP}" \
  -H "Webhook-Signature: ${WEBHOOK_SIGNATURE}" \
  --data-binary "@${BODY_FILE}" \
  http://localhost:8080
