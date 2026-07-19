#!/usr/bin/env bash
set -euo pipefail
api="${__ENV_PREFIX___API_URL:-http://localhost:8080}"
issuer="${__ENV_PREFIX___OIDC_ISSUER:-http://localhost:5556/dex}"
client="${__ENV_PREFIX___WEB_OIDC_CLIENT_ID:-__APP_SLUG__-web}"
token_body="$(curl --fail --silent --show-error -X POST "$issuer/token" -H 'Content-Type: application/x-www-form-urlencoded' --data-urlencode grant_type=password --data-urlencode username=developer@example.com --data-urlencode password=password --data-urlencode scope='openid profile email' --data-urlencode client_id="$client")"
id_token="$(printf '%s' "$token_body" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id_token"])')"
session_body="$(curl --fail --silent --show-error -X POST "$api/v1/sessions" -H 'Content-Type: application/json' --data "$(python3 -c 'import json,sys; print(json.dumps({"identityToken":sys.argv[1]}))' "$id_token")")"
session="$(printf '%s' "$session_body" | python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["token"])')"
for item in "First example" "Second example" "Third example"; do
  key="seed-$(printf '%s' "$item" | cksum | awk '{print $1}')-0000000000000000"
  curl --fail --silent --show-error -X POST "$api/v1/examples" -H "Authorization: Bearer $session" -H "Idempotency-Key: $key" -H 'Content-Type: application/json' --data "$(python3 -c 'import json,sys; print(json.dumps({"name":sys.argv[1]}))' "$item")" >/dev/null
done
echo "seeded three example resources through the public API"
