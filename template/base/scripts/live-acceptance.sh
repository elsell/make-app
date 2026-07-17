#!/usr/bin/env bash
set -euo pipefail

app_dir="${1:?generated app directory is required}"
cd "$app_dir"
export MAKE_APP_UID="${MAKE_APP_UID:-$(id -u)}"
export MAKE_APP_GID="${MAKE_APP_GID:-$(id -g)}"
docker run --rm --user 1001:1001 \
  -e HOME=/tmp/make-app-home \
  -e XDG_CACHE_HOME=/tmp/make-app-cache \
  -e COREPACK_HOME=/tmp/make-app-corepack \
  node:24.4.1-alpine3.22@sha256:820e86612c21d0636580206d802a726f2595366e1b867e564cbc652024151e8a \
  sh -lc 'corepack pnpm@11.0.7 --version' | grep -qx 11.0.7
pkce_dir=""
cleanup() {
  if [[ -n "$pkce_dir" ]]; then rm -rf "$pkce_dir"; fi
  docker compose down --volumes --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT
docker compose up -d postgres
for _ in $(seq 1 60); do
  postgres_id="$(docker compose ps -q postgres)"
  if [[ -n "$postgres_id" && "$(docker inspect -f '{{.State.Health.Status}}' "$postgres_id")" == "healthy" ]]; then break; fi
  sleep 1
done
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app -d app < apps/api/internal/adapters/dbmigrations/000001_baseline.up.sql
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app -d app <<'SQL'
CREATE TABLE schema_migrations (version bigint NOT NULL PRIMARY KEY, dirty boolean NOT NULL);
INSERT INTO schema_migrations(version, dirty) VALUES (1, false);
INSERT INTO user_models(id, email, display_name, created_at, updated_at) VALUES ('migration-marker', 'marker@example.com', 'Marker', now(), now());
INSERT INTO resource_models(id, domain, owner_user_id, name) VALUES ('migration-resource', 'example', 'migration-marker', 'Migrated resource');
SQL
docker compose run --rm --build app-migrate
docker compose exec -T postgres psql -At -U app -d app -c "SELECT id FROM user_models WHERE id='migration-marker'" | grep -qx migration-marker
docker compose exec -T postgres psql -At -U app -d app -c "SELECT to_regclass('authorization_resource_lock_models')" | grep -qx authorization_resource_lock_models
docker compose exec -T postgres psql -At -U app -d app -c "SELECT column_name FROM information_schema.columns WHERE table_name='resource_models' AND column_name='created_at'" | grep -qx created_at
docker compose exec -T postgres psql -At -U app -d app -c "SELECT created_at IS NOT NULL FROM resource_models WHERE id='migration-resource'" | grep -qx t
docker compose up -d --build spicedb dex api web

for _ in $(seq 1 180); do
  if curl -fsS http://localhost:8080/healthz >/dev/null && curl -fsS http://localhost:5556/dex/.well-known/openid-configuration >/dev/null; then break; fi
  sleep 1
done
curl -fsS http://localhost:8080/healthz >/dev/null
for _ in $(seq 1 600); do
  if curl -fsS http://localhost:5173 >/dev/null; then break; fi
  sleep 1
done
curl -fsS http://localhost:5173 >/dev/null
go test ./apps/api/internal/adapters/gormstore -run 'TestAuthorization|TestHealth' -count=1 -args -database-dsn='postgres://app:app@localhost:5432/app?sslmode=disable'
go test ./apps/api/internal/adapters/spicedb -run TestHealthRejectsLiveSchemaDrift -count=1 -args -spicedb-endpoint=localhost:50051 -spicedb-token=local-development-only-change-me -spicedb-insecure=true

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

docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app -d app -c "UPDATE schema_migrations SET dirty=true"
expect_status 503 http://localhost:8080/healthz
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app -d app -c "UPDATE schema_migrations SET dirty=false"
curl -fsS http://localhost:8080/healthz >/dev/null
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app -d app -c "UPDATE schema_migrations SET version=2"
expect_status 503 http://localhost:8080/healthz
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app -d app -c "UPDATE schema_migrations SET version=3"
curl -fsS http://localhost:8080/healthz >/dev/null

owner_token="$(token __APP_SLUG__-web developer@example.com)"
mobile_token="$(token __APP_SLUG__-mobile developer@example.com)"
docs_token="$(token __APP_SLUG__-docs developer@example.com)"
second_token="$(token __APP_SLUG__-web second@example.com)"
wrong_audience_token="$(token __APP_SLUG__-wrong-audience developer@example.com)"
IFS=. read -r token_header token_payload token_signature <<<"$owner_token"
replacement=A
[[ "${token_signature:0:1}" == A ]] && replacement=B
tampered_token="$token_header.$token_payload.$replacement${token_signature:1}"

expect_status 401 http://localhost:8080/v1/me
expect_status 401 -H 'Authorization: Bearer malformed' http://localhost:8080/v1/me
expect_status 401 -H "Authorization: Bearer $tampered_token" http://localhost:8080/v1/me
expect_status 401 -H "Authorization: Bearer $wrong_audience_token" http://localhost:8080/v1/me
me="$(curl -fsS -H "Authorization: Bearer $owner_token" http://localhost:8080/v1/me)"
owner_id="$(printf '%s' "$me" | python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["id"])')"
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -v "owner_id=$owner_id" -U app -d app <<'SQL'
UPDATE resource_models SET owner_user_id=:'owner_id' WHERE id='migration-resource';
SQL
curl -fsS -H "Authorization: Bearer $mobile_token" http://localhost:8080/v1/me >/dev/null
curl -fsS -H "Authorization: Bearer $docs_token" http://localhost:8080/v1/me >/dev/null
curl -fsS http://localhost:8080/openapi.json | python3 -c 'import json,sys;s=json.load(sys.stdin)["components"]["securitySchemes"]["oidc"];f=s["flows"]["authorizationCode"];assert f["x-scalar-client-id"]=="__APP_SLUG__-docs";assert f["x-usePkce"]=="SHA-256";assert f["x-scalar-redirect-uri"]=="http://localhost:8080/docs"'
curl -fsS http://localhost:8080/docs | grep -q '@scalar/api-reference'
curl -fsS http://localhost:8080/docs | grep -q '"selectedScopes":\["openid","profile","email"\]'
curl -fsS -D - -o /dev/null http://localhost:8080/docs | grep -i "content-security-policy:.*connect-src 'self'"
docs_discovery="$(curl -fsS http://localhost:8080/oidc/.well-known/openid-configuration)"
printf '%s' "$docs_discovery" | python3 -c 'import json,sys;d=json.load(sys.stdin);assert d["token_endpoint"]=="http://localhost:8080/oidc/token";assert d["authorization_endpoint"].startswith("http://localhost:5556/")'

pkce_verifier='make-app-live-acceptance-verifier-0123456789abcdefghijklmnop'
pkce_challenge="$(PKCE_VERIFIER="$pkce_verifier" python3 -c 'import base64,hashlib,os;print(base64.urlsafe_b64encode(hashlib.sha256(os.environ["PKCE_VERIFIER"].encode()).digest()).rstrip(b"=").decode())')"
pkce_state='make-app-live-acceptance-state'
pkce_dir="$(mktemp -d)"
authorization_url="$(printf '%s' "$docs_discovery" | python3 -c 'import json,sys;print(json.load(sys.stdin)["authorization_endpoint"])')"
curl -fsS -L -c "$pkce_dir/cookies" -b "$pkce_dir/cookies" --get "$authorization_url" \
  --data-urlencode client_id=__APP_SLUG__-docs --data-urlencode redirect_uri=http://localhost:8080/docs \
  --data-urlencode response_type=code --data-urlencode 'scope=openid profile email' \
  --data-urlencode "state=$pkce_state" --data-urlencode "code_challenge=$pkce_challenge" \
  --data-urlencode code_challenge_method=S256 -o "$pkce_dir/login.html"
login_action="$(LOGIN_HTML="$pkce_dir/login.html" python3 -c 'import html,os,re;p=open(os.environ["LOGIN_HTML"]).read();print(html.unescape(re.search(r"<form[^>]+action=\"([^\"]+)",p).group(1)))')"
curl -sS -D "$pkce_dir/headers" -o /dev/null -c "$pkce_dir/cookies" -b "$pkce_dir/cookies" \
  -X POST --data-urlencode login=developer@example.com --data-urlencode password=password "http://localhost:5556$login_action"
pkce_redirect="$(sed -n 's/^Location: //p' "$pkce_dir/headers" | tr -d '\r')"
pkce_code="$(PKCE_REDIRECT="$pkce_redirect" PKCE_STATE="$pkce_state" python3 -c 'import os,urllib.parse as u;q=u.parse_qs(u.urlparse(os.environ["PKCE_REDIRECT"]).query);assert q["state"]==[os.environ["PKCE_STATE"]];print(q["code"][0])')"
docs_pkce_token="$(curl -fsS -X POST http://localhost:8080/oidc/token \
  --data-urlencode grant_type=authorization_code --data-urlencode "code=$pkce_code" \
  --data-urlencode "code_verifier=$pkce_verifier" --data-urlencode client_id=__APP_SLUG__-docs \
  --data-urlencode redirect_uri=http://localhost:8080/docs | python3 -c 'import json,sys;print(json.load(sys.stdin)["access_token"])')"
curl -fsS -H "Authorization: Bearer $docs_pkce_token" http://localhost:8080/v1/me >/dev/null
node scripts/scalar-browser-acceptance.mjs
docs_created="$(curl -fsS -X POST -H "Authorization: Bearer $docs_pkce_token" -H 'Content-Type: application/json' -d '{"name":"Docs PKCE resource"}' http://localhost:8080/v1/examples)"
docs_resource_id="$(printf '%s' "$docs_created" | python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["id"])')"
curl -fsS -H "Authorization: Bearer $docs_pkce_token" "http://localhost:8080/v1/examples/$docs_resource_id" >/dev/null
curl -fsS -H "Authorization: Bearer $docs_pkce_token" http://localhost:8080/v1/examples | grep -q "$docs_resource_id"
rm -rf "$pkce_dir"
pkce_dir=""

for name in Page-A Page-B Page-C; do
  curl -fsS -X POST -H "Authorization: Bearer $owner_token" -H 'Content-Type: application/json' -d "{\"name\":\"$name\"}" http://localhost:8080/v1/examples >/dev/null
done
first_page="$(curl -fsS -H "Authorization: Bearer $owner_token" 'http://localhost:8080/v1/examples?limit=2')"
cursor="$(printf '%s' "$first_page" | python3 -c 'import json,sys;r=json.load(sys.stdin);assert len(r["data"])==2;print(r["meta"]["nextCursor"])')"
curl -fsS -X POST -H "Authorization: Bearer $owner_token" -H 'Content-Type: application/json' -d '{"name":"Page-Late"}' http://localhost:8080/v1/examples >/dev/null
second_page="$(curl -fsS -H "Authorization: Bearer $owner_token" "http://localhost:8080/v1/examples?limit=2&cursor=$cursor")"
FIRST_PAGE="$first_page" SECOND_PAGE="$second_page" python3 -c 'import json,os;a=json.loads(os.environ["FIRST_PAGE"]);b=json.loads(os.environ["SECOND_PAGE"]);rows=a["data"]+b["data"];assert len(rows)==4 and len({v["id"] for v in rows})==4 and any(v["name"]=="Migrated resource" for v in rows) and all(v["name"]!="Page-Late" for v in rows)'
expect_status 400 -H "Authorization: Bearer $owner_token" 'http://localhost:8080/v1/examples?cursor=not-a-signed-cursor'
expect_status 400 -H "Authorization: Bearer $second_token" "http://localhost:8080/v1/examples?cursor=$cursor"
forged_cursor="A${cursor:1}"
expect_status 400 -H "Authorization: Bearer $owner_token" "http://localhost:8080/v1/examples?cursor=$forged_cursor"
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

docker compose stop spicedb
expect_status 503 http://localhost:8080/healthz
docker compose start spicedb
for _ in $(seq 1 60); do curl -fsS http://localhost:8080/healthz >/dev/null 2>&1 && break; sleep 1; done
curl -fsS http://localhost:8080/healthz >/dev/null

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
owner_token="$(token __APP_SLUG__-web developer@example.com)"
curl -fsS -H "Authorization: Bearer $owner_token" "http://localhost:8080/v1/examples/$persistent_id" >/dev/null

echo "live authentication and authorization acceptance passed"
