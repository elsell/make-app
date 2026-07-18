#!/usr/bin/env bash
set -euo pipefail

app_dir="${1:?generated app directory is required}"
cd "$app_dir"
lock_dir="${TMPDIR:-/tmp}/__APP_SLUG__-live-acceptance.lock"
lock_acquired=""
for _ in $(seq 1 1200); do
  if mkdir "$lock_dir" 2>/dev/null; then printf '%s\n' "$$" > "$lock_dir/pid"; lock_acquired=1; break; fi
  lock_pid="$(cat "$lock_dir/pid" 2>/dev/null || true)"
  if [[ -n "$lock_pid" ]] && ! kill -0 "$lock_pid" 2>/dev/null; then rm -rf "$lock_dir"; continue; fi
  sleep 1
done
[[ -n "$lock_acquired" ]] || { echo "timed out waiting for the live acceptance port lock" >&2; exit 1; }
export MAKE_APP_UID="${MAKE_APP_UID:-$(id -u)}"
export MAKE_APP_GID="${MAKE_APP_GID:-$(id -g)}"
export COMPOSE_PROJECT_NAME="make-app-acceptance-${MAKE_APP_ACCEPTANCE_RUN_ID:-$$}"
pkce_dir=""
cleanup() {
  if [[ -n "$pkce_dir" ]]; then rm -rf "$pkce_dir"; fi
  docker compose down --volumes --remove-orphans >/dev/null 2>&1 || true
	rm -rf "$lock_dir"
}
trap cleanup EXIT
docker run --rm --user 1001:1001 \
  -e HOME=/tmp/make-app-home \
  -e XDG_CACHE_HOME=/tmp/make-app-cache \
  -e COREPACK_HOME=/tmp/make-app-corepack \
  node:24.4.1-alpine3.22@sha256:820e86612c21d0636580206d802a726f2595366e1b867e564cbc652024151e8a \
  sh -lc 'corepack pnpm@11.0.7 --version' | grep -qx 11.0.7
docker compose down --volumes --remove-orphans >/dev/null 2>&1 || true
docker compose up -d postgres
for _ in $(seq 1 60); do
  postgres_id="$(docker compose ps -q postgres)"
  if [[ -n "$postgres_id" && "$(docker inspect -f '{{.State.Health.Status}}' "$postgres_id")" == "healthy" ]]; then break; fi
  sleep 1
done
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app_migrator -d app < apps/api/internal/adapters/dbmigrations/000001_baseline.up.sql
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app_migrator -d app <<'SQL'
CREATE TABLE schema_migrations (version bigint NOT NULL PRIMARY KEY, dirty boolean NOT NULL);
INSERT INTO schema_migrations(version, dirty) VALUES (1, false);
INSERT INTO user_models(id, email, display_name, created_at, updated_at) VALUES ('migration-marker', 'marker@example.com', 'Marker', now(), now());
INSERT INTO resource_models(id, domain, owner_user_id, name) VALUES ('migration-resource', 'example', 'migration-marker', 'Migrated resource');
SQL
for migration in apps/api/internal/adapters/dbmigrations/00000{2..8}_*.up.sql; do
  docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app_migrator -d app < "$migration"
  version="$(basename "$migration" | cut -d_ -f1 | sed 's/^0*//')"
  docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app_migrator -d app -c "UPDATE schema_migrations SET version=$version"
done
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
latest_migration="$(ls apps/api/internal/adapters/dbmigrations/[0-9][0-9][0-9][0-9][0-9][0-9]_*.up.sql | sort | tail -n1)"
latest_version="$(basename "$latest_migration" | cut -d_ -f1 | sed 's/^0*//')"
docker compose stop api
go test ./apps/api/internal/adapters/gormstore -count=1 -args -database-dsn='postgres://app_migrator:app_migrator@localhost:5432/app?sslmode=disable'
for package in $(go list ./apps/api/internal/adapters/gormstore/... | grep -v '/gormstore$'); do
  go test "$package" -count=1 -args -database-dsn='postgres://app_migrator:app_migrator@localhost:5432/app?sslmode=disable'
done
docker compose up -d api
for _ in $(seq 1 60); do
  if curl -fsS http://localhost:8080/healthz >/dev/null; then break; fi
  sleep 1
done
curl -fsS http://localhost:8080/healthz >/dev/null
go test ./apps/api/internal/adapters/spicedb -run TestRuntimeCredentialReadsRequiredSchemaButCannotRewriteIt -count=1 -args -spicedb-endpoint=localhost:50051 -spicedb-token=local-development-runtime-change-me -spicedb-insecure=true

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

docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app_migrator -d app -c "UPDATE schema_migrations SET dirty=true"
expect_status 503 http://localhost:8080/healthz
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app_migrator -d app -c "UPDATE schema_migrations SET dirty=false"
curl -fsS http://localhost:8080/healthz >/dev/null
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app_migrator -d app -c "UPDATE schema_migrations SET version=2"
expect_status 503 http://localhost:8080/healthz
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app_migrator -d app -c "UPDATE schema_migrations SET version=$latest_version"
curl -fsS http://localhost:8080/healthz >/dev/null

session() {
  identity_token="$(token "$1" "$2")"
  IDENTITY_TOKEN="$identity_token" python3 -c 'import json,os;print(json.dumps({"identityToken":os.environ["IDENTITY_TOKEN"]}))' |
    curl -fsS -X POST -H 'Content-Type: application/json' --data-binary @- http://localhost:8080/v1/sessions |
    python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["token"])'
}
owner_oidc_token="$(token __APP_SLUG__-web developer@example.com)"
owner_token="$(session __APP_SLUG__-web developer@example.com)"
mobile_token="$(session __APP_SLUG__-mobile developer@example.com)"
docs_token="$(session __APP_SLUG__-docs developer@example.com)"
second_token="$(session __APP_SLUG__-web second@example.com)"
refreshed="$(curl -fsS -X POST -H "Authorization: Bearer $owner_token" http://localhost:8080/v1/session/refresh)"
refreshed_owner_token="$(printf '%s' "$refreshed" | python3 -c 'import json,sys;r=json.load(sys.stdin)["data"];assert r["expiresAt"];print(r["token"])')"
expect_status 401 -H "Authorization: Bearer $owner_token" http://localhost:8080/v1/me
owner_token="$refreshed_owner_token"
wrong_audience_token="$(token __APP_SLUG__-wrong-audience developer@example.com)"
IFS=. read -r token_header token_payload token_signature <<<"$owner_oidc_token"
replacement=A
[[ "${token_signature:0:1}" == A ]] && replacement=B
tampered_token="$token_header.$token_payload.$replacement${token_signature:1}"

expect_status 401 http://localhost:8080/v1/me
expect_status 401 http://localhost:8080/v1/audit-events
expect_status 401 -H 'Authorization: Bearer malformed' http://localhost:8080/v1/me
expect_status 401 -H "Authorization: Bearer $tampered_token" http://localhost:8080/v1/me
expect_status 401 -H "Authorization: Bearer $wrong_audience_token" http://localhost:8080/v1/me
tampered_body="$(IDENTITY_TOKEN="$tampered_token" python3 -c 'import json,os;print(json.dumps({"identityToken":os.environ["IDENTITY_TOKEN"]}))')"
wrong_audience_body="$(IDENTITY_TOKEN="$wrong_audience_token" python3 -c 'import json,os;print(json.dumps({"identityToken":os.environ["IDENTITY_TOKEN"]}))')"
expect_status 401 -X POST -H 'Content-Type: application/json' -d "$tampered_body" http://localhost:8080/v1/sessions
expect_status 401 -X POST -H 'Content-Type: application/json' -d "$wrong_audience_body" http://localhost:8080/v1/sessions
me="$(curl -fsS -H "Authorization: Bearer $owner_token" http://localhost:8080/v1/me)"
owner_id="$(printf '%s' "$me" | python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["id"])')"
second_id="$(curl -fsS -H "Authorization: Bearer $second_token" http://localhost:8080/v1/me | python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["id"])')"
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
docs_created="$(curl -fsS -X POST -H "Authorization: Bearer $docs_pkce_token" -H 'Idempotency-Key: docs-pkce-create-0001' -H 'Content-Type: application/json' -d '{"name":"Docs PKCE resource"}' http://localhost:8080/v1/examples)"
docs_resource_id="$(printf '%s' "$docs_created" | python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["id"])')"
curl -fsS -H "Authorization: Bearer $docs_pkce_token" "http://localhost:8080/v1/examples/$docs_resource_id" >/dev/null
curl -fsS -H "Authorization: Bearer $docs_pkce_token" http://localhost:8080/v1/examples | grep -q "$docs_resource_id"
rm -rf "$pkce_dir"
pkce_dir=""

for name in Page-A Page-B Page-C; do
  curl -fsS -X POST -H "Authorization: Bearer $owner_token" -H "Idempotency-Key: page-create-$name-0001" -H 'Content-Type: application/json' -d "{\"name\":\"$name\"}" http://localhost:8080/v1/examples >/dev/null
done
first_page="$(curl -fsS -H "Authorization: Bearer $owner_token" 'http://localhost:8080/v1/examples?limit=2')"
cursor="$(printf '%s' "$first_page" | python3 -c 'import json,sys;r=json.load(sys.stdin);assert len(r["data"])==2;print(r["meta"]["nextCursor"])')"
curl -fsS -X POST -H "Authorization: Bearer $owner_token" -H 'Idempotency-Key: page-late-create-0001' -H 'Content-Type: application/json' -d '{"name":"Page-Late"}' http://localhost:8080/v1/examples >/dev/null
second_page="$(curl -fsS -H "Authorization: Bearer $owner_token" "http://localhost:8080/v1/examples?limit=2&cursor=$cursor")"
FIRST_PAGE="$first_page" SECOND_PAGE="$second_page" python3 -c 'import json,os;a=json.loads(os.environ["FIRST_PAGE"]);b=json.loads(os.environ["SECOND_PAGE"]);rows=a["data"]+b["data"];assert len(rows)==4 and len({v["id"] for v in rows})==4 and any(v["name"]=="Migrated resource" for v in rows) and all(v["name"]!="Page-Late" for v in rows)'
expect_status 400 -H "Authorization: Bearer $owner_token" 'http://localhost:8080/v1/examples?cursor=not-a-signed-cursor'
expect_status 400 -H "Authorization: Bearer $second_token" "http://localhost:8080/v1/examples?cursor=$cursor"
forged_cursor="A${cursor:1}"
expect_status 400 -H "Authorization: Bearer $owner_token" "http://localhost:8080/v1/examples?cursor=$forged_cursor"
expect_status 422 -H "Authorization: Bearer $owner_token" 'http://localhost:8080/v1/examples?limit=101'

created="$(curl -fsS -X POST -H "Authorization: Bearer $owner_token" -H 'Idempotency-Key: private-create-0001' -H 'Content-Type: application/json' -d '{"name":"Private resource"}' http://localhost:8080/v1/examples)"
replayed="$(curl -fsS -X POST -H "Authorization: Bearer $owner_token" -H 'Idempotency-Key: private-create-0001' -H 'Content-Type: application/json' -d '{"name":"Private resource"}' http://localhost:8080/v1/examples)"
[[ "$(printf '%s' "$replayed" | python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["id"])')" == "$(printf '%s' "$created" | python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["id"])')" ]]
expect_status 409 -X POST -H "Authorization: Bearer $owner_token" -H 'Idempotency-Key: private-create-0001' -H 'Content-Type: application/json' -d '{"name":"Different resource"}' http://localhost:8080/v1/examples
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

owner_audit="$(curl -fsS -H "Authorization: Bearer $owner_token" http://localhost:8080/v1/audit-events)"
OWNER_AUDIT="$owner_audit" RESOURCE_ID="$resource_id" python3 -c 'import json,os;events=json.loads(os.environ["OWNER_AUDIT"])["data"];actions={e["action"] for e in events if e["targetId"]==os.environ["RESOURCE_ID"]};required={"resource.created","resource.viewed","resource.updated","resource.deleted"};assert required <= actions,(required,actions)'
second_audit="$(curl -fsS -H "Authorization: Bearer $second_token" http://localhost:8080/v1/audit-events)"
SECOND_AUDIT="$second_audit" RESOURCE_ID="$resource_id" python3 -c 'import json,os;events=json.loads(os.environ["SECOND_AUDIT"])["data"];matching=[e for e in events if e["targetId"]==os.environ["RESOURCE_ID"]];assert len(matching)==3 and all(e["action"]=="resource.access_denied" and e["outcome"]=="denied" for e in matching),matching'
if docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app -d app -c "UPDATE schema_migrations SET dirty=true"; then
  echo "runtime database role mutated migration ledger" >&2; exit 1
fi
if docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app_migrator -d app -c "TRUNCATE audit_event_models"; then
  echo "database owner bypassed append-only audit truncation guard" >&2; exit 1
fi
audit_first="$(curl -fsS -H "Authorization: Bearer $owner_token" 'http://localhost:8080/v1/audit-events?limit=2')"
audit_cursor="$(printf '%s' "$audit_first" | python3 -c 'import json,sys;r=json.load(sys.stdin);assert len(r["data"])==2;print(r["meta"]["nextCursor"])')"
audit_second="$(curl -fsS -H "Authorization: Bearer $owner_token" "http://localhost:8080/v1/audit-events?limit=2&cursor=$audit_cursor")"
AUDIT_FIRST="$audit_first" AUDIT_SECOND="$audit_second" python3 -c 'import json,os;a=json.loads(os.environ["AUDIT_FIRST"])["data"];b=json.loads(os.environ["AUDIT_SECOND"])["data"];assert not ({e["id"] for e in a}&{e["id"] for e in b})'
expect_status 400 -H "Authorization: Bearer $second_token" "http://localhost:8080/v1/audit-events?limit=2&cursor=$audit_cursor"

second_oidc_token="$(token __APP_SLUG__-web second@example.com)"
expect_status 204 -X DELETE -H "Authorization: Bearer $second_token" http://localhost:8080/v1/me
expect_status 401 -H "Authorization: Bearer $second_token" http://localhost:8080/v1/me
second_oidc_body="$(IDENTITY_TOKEN="$second_oidc_token" python3 -c 'import json,os;print(json.dumps({"identityToken":os.environ["IDENTITY_TOKEN"]}))')"
expect_status 401 -X POST -H 'Content-Type: application/json' -d "$second_oidc_body" http://localhost:8080/v1/sessions
docker compose exec -T postgres psql -At -U app -d app -v "second_id=$second_id" <<'AUDIT_SQL' | grep -qx 1
SELECT count(*) FROM audit_event_models WHERE owner_user_id=:'second_id' AND action='user.deactivated';
AUDIT_SQL

docker compose stop spicedb
expect_status 503 http://localhost:8080/healthz
docker compose start spicedb
for _ in $(seq 1 60); do curl -fsS http://localhost:8080/healthz >/dev/null 2>&1 && break; sleep 1; done
curl -fsS http://localhost:8080/healthz >/dev/null

persistent="$(curl -fsS -X POST -H "Authorization: Bearer $owner_token" -H 'Idempotency-Key: restart-create-0001' -H 'Content-Type: application/json' -d '{"name":"Restart survivor"}' http://localhost:8080/v1/examples)"
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
owner_token="$(session __APP_SLUG__-web developer@example.com)"
curl -fsS -H "Authorization: Bearer $owner_token" "http://localhost:8080/v1/examples/$persistent_id" >/dev/null
curl -fsS -H "Authorization: Bearer $owner_token" http://localhost:8080/v1/audit-events | PERSISTENT_ID="$persistent_id" python3 -c 'import json,os,sys;assert any(e["action"]=="resource.created" and e["targetId"]==os.environ["PERSISTENT_ID"] for e in json.load(sys.stdin)["data"])'

echo "live authentication, authorization, and audit acceptance passed"
