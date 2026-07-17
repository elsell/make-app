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
go test ./apps/api/internal/adapters/gormstore -run TestAuthorizationOutboxLeasesAreOwnedRecoverableAndFIFO -count=1 -args -database-dsn='postgres://app:app@localhost:5432/app?sslmode=disable'

token() {
  curl -fsS -X POST http://localhost:5556/dex/token \
    -d grant_type=password -d "client_id=$1" -d "username=$2" -d password=password \
    -d scope="openid profile email" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id_token"])'
}
expect_status() {
  expected="$1"; shift
  actual="$(curl -sS -o /dev/null -w '%{http_code}' "$@")"
  [[ "$actual" == "$expected" ]] || { echo "expected HTTP $expected, got $actual: $*" >&2; return 1; }
}

owner_token="$(token secure-app-web developer@example.com)"
mobile_token="$(token secure-app-mobile developer@example.com)"
docs_token="$(token secure-app-docs developer@example.com)"
second_token="$(token secure-app-web second@example.com)"
wrong_audience_token="$(token secure-app-wrong-audience developer@example.com)"
IFS=. read -r token_header token_payload token_signature <<<"$owner_token"
replacement=A
[[ "${token_signature:0:1}" == A ]] && replacement=B
tampered_token="$token_header.$token_payload.$replacement${token_signature:1}"

expect_status 401 http://localhost:8080/v1/me
expect_status 401 -H 'Authorization: Bearer malformed' http://localhost:8080/v1/me
expect_status 401 -H "Authorization: Bearer $tampered_token" http://localhost:8080/v1/me
expect_status 401 -H "Authorization: Bearer $wrong_audience_token" http://localhost:8080/v1/me
curl -fsS -H "Authorization: Bearer $owner_token" http://localhost:8080/v1/me >/dev/null
curl -fsS -H "Authorization: Bearer $mobile_token" http://localhost:8080/v1/me >/dev/null
curl -fsS -H "Authorization: Bearer $docs_token" http://localhost:8080/v1/me >/dev/null
curl -fsS http://localhost:8080/openapi.json | python3 -c 'import json,sys;s=json.load(sys.stdin)["components"]["securitySchemes"]["oidc"];assert s["x-scalar-client-id"]=="secure-app-docs";assert s["x-usePkce"]=="SHA-256"'
curl -fsS http://localhost:8080/docs | grep -q '@scalar/api-reference'

for name in Page-A Page-B Page-C; do
  curl -fsS -X POST -H "Authorization: Bearer $owner_token" -H 'Content-Type: application/json' -d "{\"name\":\"$name\"}" http://localhost:8080/v1/examples >/dev/null
done
first_page="$(curl -fsS -H "Authorization: Bearer $owner_token" 'http://localhost:8080/v1/examples?limit=2')"
cursor="$(printf '%s' "$first_page" | python3 -c 'import json,sys;r=json.load(sys.stdin);assert len(r["data"])==2;print(r["meta"]["nextCursor"])')"
second_page="$(curl -fsS -H "Authorization: Bearer $owner_token" "http://localhost:8080/v1/examples?limit=2&cursor=$cursor")"
FIRST_PAGE="$first_page" SECOND_PAGE="$second_page" python3 -c 'import json,os;a=json.loads(os.environ["FIRST_PAGE"]);b=json.loads(os.environ["SECOND_PAGE"]);x={v["id"] for v in a["data"]};y={v["id"] for v in b["data"]};assert y and not x.intersection(y)'
expect_status 400 -H "Authorization: Bearer $owner_token" 'http://localhost:8080/v1/examples?cursor=%%%'
expect_status 422 -H "Authorization: Bearer $owner_token" 'http://localhost:8080/v1/examples?limit=101'

created="$(curl -fsS -X POST -H "Authorization: Bearer $owner_token" -H 'Content-Type: application/json' -d '{"name":"Private resource"}' http://localhost:8080/v1/examples)"
resource_id="$(printf '%s' "$created" | python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["id"])')"
curl -fsS -H "Authorization: Bearer $owner_token" "http://localhost:8080/v1/examples/$resource_id" >/dev/null
expect_status 404 -H "Authorization: Bearer $second_token" "http://localhost:8080/v1/examples/$resource_id"
expect_status 404 -X PATCH -H "Authorization: Bearer $second_token" -H 'Content-Type: application/json' -d '{"name":"stolen"}' "http://localhost:8080/v1/examples/$resource_id"
expect_status 404 -X DELETE -H "Authorization: Bearer $second_token" "http://localhost:8080/v1/examples/$resource_id"
if curl -fsS -H "Authorization: Bearer $second_token" http://localhost:8080/v1/examples | grep -q "$resource_id"; then
  echo "cross-user resource leaked through list" >&2; exit 1
fi

updated="$(curl -fsS -X PATCH -H "Authorization: Bearer $owner_token" -H 'Content-Type: application/json' -d '{"name":"Updated resource"}' "http://localhost:8080/v1/examples/$resource_id")"
printf '%s' "$updated" | python3 -c 'import json,sys; assert json.load(sys.stdin)["data"]["name"] == "Updated resource"'
expect_status 204 -X DELETE -H "Authorization: Bearer $owner_token" "http://localhost:8080/v1/examples/$resource_id"
expect_status 404 -H "Authorization: Bearer $owner_token" "http://localhost:8080/v1/examples/$resource_id"
if curl -fsS -H "Authorization: Bearer $owner_token" http://localhost:8080/v1/examples | grep -q "$resource_id"; then
  echo "deleted resource remained discoverable" >&2; exit 1
fi

persistent="$(curl -fsS -X POST -H "Authorization: Bearer $owner_token" -H 'Content-Type: application/json' -d '{"name":"Restart survivor"}' http://localhost:8080/v1/examples)"
persistent_id="$(printf '%s' "$persistent" | python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["id"])')"
docker compose stop api spicedb dex postgres
docker compose start postgres
for _ in $(seq 1 60); do
  if docker compose exec -T postgres pg_isready -U app -d app >/dev/null 2>&1; then break; fi
  sleep 1
done
docker compose up -d spicedb-migrate spicedb dex api
for _ in $(seq 1 180); do
  if curl -fsS http://localhost:8080/healthz >/dev/null && curl -fsS http://localhost:5556/dex/.well-known/openid-configuration >/dev/null; then break; fi
  sleep 1
done
owner_token="$(token secure-app-web developer@example.com)"
curl -fsS -H "Authorization: Bearer $owner_token" "http://localhost:8080/v1/examples/$persistent_id" >/dev/null

echo "live authentication and authorization acceptance passed"
