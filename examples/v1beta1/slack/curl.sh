# Slack signs each request with a timestamp, so the signature must be generated at
# request time rather than hardcoded: a stale signature is rejected once the
# X-Slack-Request-Timestamp falls outside the interceptor's five minute window.
BODY='token=EidhofDor5uIpqQ9RrtOVdnC&team_id=T04PK47eDS4&team_domain=demoworkspace-tid8978&channel_id=C04NETw4NBH&channel_name=sample-app&user_id=U04NVDwF7R8&command=%2Fbuild&text=main+2222&api_app_id=A04NXU23L&is_enterprise_install=false&response_url=https%3A%2F%2Fhooks.slack.com%2Fcommands%2FT04PK47EDS4%2F4863712501879%2FdOMNffCDfTjlSskBrmB1bOtR&trigger_id=4890883491553.4801143489888.910b8eaae200b381834de25310583f74'
TIMESTAMP=$(date +%s)
SIGNING_SECRET=$(kubectl get secret slack-secret -o jsonpath='{.data.signingSecret}' | base64 -d)
SIGNATURE="v0=$(printf 'v0:%s:%s' "$TIMESTAMP" "$BODY" | openssl dgst -sha256 -hmac "$SIGNING_SECRET" -hex | sed 's/^.* //')"

curl -v \
-H "X-Slack-Request-Timestamp: $TIMESTAMP" \
-H "X-Slack-Signature: $SIGNATURE" \
-H 'Content-Type: application/x-www-form-urlencoded' \
--data-raw "$BODY" \
http://localhost:8080
