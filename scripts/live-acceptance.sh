#!/usr/bin/env bash
set -euo pipefail

app_dir="${1:?generated app directory is required}"
cd "$app_dir"
docker compose up -d --build postgres spicedb dex api

for _ in $(seq 1 180); do
  if curl -fsS http://localhost:8080/healthz >/dev/null && curl -fsS http://localhost:5556/dex/.well-known/openid-configuration >/dev/null; then break; fi
  sleep 1
done
curl -fsS http://localhost:8080/healthz >/dev/null

token() {
  curl -fsS -X POST http://localhost:5556/dex/token \
    -d grant_type=password -d "client_id=$1" -d "username=$2" -d password=password \
    -d scope="openid profile email" | python3 -c 'import json,sys; print(json.load(sys.stdin)["access_token"])'
}
expect_status() {
  expected="$1"; shift
  actual="$(curl -sS -o /dev/null -w '%{http_code}' "$@")"
  [[ "$actual" == "$expected" ]] || { echo "expected HTTP $expected, got $actual: $*" >&2; return 1; }
}

owner_token="$(token secure-app-web developer@example.com)"
second_token="$(token secure-app-web second@example.com)"
wrong_audience_token="$(token secure-app-wrong-audience developer@example.com)"

expect_status 401 http://localhost:8080/v1/me
expect_status 401 -H 'Authorization: Bearer malformed' http://localhost:8080/v1/me
expect_status 401 -H "Authorization: Bearer $wrong_audience_token" http://localhost:8080/v1/me
curl -fsS -H "Authorization: Bearer $owner_token" http://localhost:8080/v1/me >/dev/null

created="$(curl -fsS -X POST -H "Authorization: Bearer $owner_token" -H 'Content-Type: application/json' -d '{"name":"Private resource"}' http://localhost:8080/v1/examples)"
resource_id="$(printf '%s' "$created" | python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["id"])')"
for _ in $(seq 1 20); do
  if curl -fsS -H "Authorization: Bearer $owner_token" "http://localhost:8080/v1/examples/$resource_id" >/dev/null; then break; fi
  sleep 1
done
curl -fsS -H "Authorization: Bearer $owner_token" "http://localhost:8080/v1/examples/$resource_id" >/dev/null
expect_status 404 -H "Authorization: Bearer $second_token" "http://localhost:8080/v1/examples/$resource_id"
if curl -fsS -H "Authorization: Bearer $second_token" http://localhost:8080/v1/examples | grep -q "$resource_id"; then
  echo "cross-user resource leaked through list" >&2; exit 1
fi

echo "live authentication and authorization acceptance passed"
