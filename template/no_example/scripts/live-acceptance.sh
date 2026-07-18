#!/usr/bin/env bash
set -euo pipefail
app_dir="${1:?generated app directory is required}"
cd "$app_dir"
lock_dir="${TMPDIR:-/tmp}/__APP_SLUG__-live-acceptance.lock"
for _ in $(seq 1 1200); do
  if mkdir "$lock_dir" 2>/dev/null; then printf '%s\n' "$$" > "$lock_dir/pid"; acquired=1; break; fi
  lock_pid="$(cat "$lock_dir/pid" 2>/dev/null || true)"
  if [[ -n "$lock_pid" ]] && ! kill -0 "$lock_pid" 2>/dev/null; then rm -rf "$lock_dir"; continue; fi
  sleep 1
done
[[ -n "${acquired:-}" ]] || { echo "timed out waiting for the live acceptance port lock" >&2; exit 1; }
export MAKE_APP_UID="${MAKE_APP_UID:-$(id -u)}" MAKE_APP_GID="${MAKE_APP_GID:-$(id -g)}"
export COMPOSE_PROJECT_NAME="make-app-acceptance-${MAKE_APP_ACCEPTANCE_RUN_ID:-$$}"
export __ENV_PREFIX___ACCOUNT_PROVISIONING_MODE=existing
export __ENV_PREFIX___ACCOUNT_INVITED_EMAILS=developer@example.com
cleanup() { status=$?; if [[ "$status" -ne 0 ]]; then docker compose ps -a >&2 || true; docker compose logs --tail=200 >&2 || true; fi; docker compose down --volumes --remove-orphans --rmi local >/dev/null 2>&1 || true; rm -rf "$lock_dir"; return "$status"; }
trap cleanup EXIT

docker compose down --volumes --remove-orphans >/dev/null 2>&1 || true
docker compose up -d postgres
for _ in $(seq 1 60); do
  id="$(docker compose ps -q postgres)"
  [[ -n "$id" && "$(docker inspect -f '{{.State.Health.Status}}' "$id")" == healthy ]] && break
  sleep 1
done
docker compose run --rm --build app-migrate
docker compose up -d --build spicedb dex api web
for _ in $(seq 1 180); do curl -fsS http://localhost:8080/readyz >/dev/null 2>&1 && curl -fsS http://localhost:5556/dex/.well-known/openid-configuration >/dev/null 2>&1 && break; sleep 1; done
curl -fsS http://localhost:8080/healthz >/dev/null
curl -fsS http://localhost:8080/readyz >/dev/null
for _ in $(seq 1 180); do curl -fsS http://localhost:5173 >/dev/null 2>&1 && break; sleep 1; done
curl -fsS http://localhost:5173 >/dev/null

token() {
  curl -fsS -X POST http://localhost:5556/dex/token -d grant_type=password -d "client_id=$1" -d "username=$2" -d password=password -d scope='openid profile email' |
    python3 -c 'import json,sys;print(json.load(sys.stdin)["id_token"])'
}
session() {
  identity_token="$(token "$1" "$2")"
  IDENTITY_TOKEN="$identity_token" python3 -c 'import json,os;print(json.dumps({"identityToken":os.environ["IDENTITY_TOKEN"]}))' |
    curl -fsS -X POST -H 'Content-Type: application/json' --data-binary @- http://localhost:8080/v1/sessions |
    python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["token"])'
}
expect_status() { expected="$1"; shift; actual="$(curl -sS -o /dev/null -w '%{http_code}' "$@")"; [[ "$actual" == "$expected" ]] || { echo "expected HTTP $expected, got $actual: $*" >&2; return 1; }; }

owner_token="$(session __APP_SLUG__-web developer@example.com)"
expect_status 401 http://localhost:8080/v1/me
expect_status 401 -H 'Authorization: Bearer malformed' http://localhost:8080/v1/me
me="$(curl -fsS -H "Authorization: Bearer $owner_token" http://localhost:8080/v1/me)"
owner_id="$(printf '%s' "$me" | python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["id"])')"
curl -fsS -H "Authorization: Bearer $owner_token" http://localhost:8080/v1/audit-events |
  python3 -c 'import json,sys;assert any(e["action"]=="user.viewed" for e in json.load(sys.stdin)["data"])'

curl -fsS http://localhost:8080/openapi.json | python3 -c 'import json,sys;s=json.load(sys.stdin)["components"]["securitySchemes"]["oidc"];assert s["flows"]["authorizationCode"]["x-usePkce"]=="SHA-256"'
curl -fsS http://localhost:8080/docs | grep -q '@scalar/api-reference'
discovery="$(curl -fsS http://localhost:8080/oidc/.well-known/openid-configuration)"
printf '%s' "$discovery" | python3 -c 'import json,sys;d=json.load(sys.stdin);assert d["token_endpoint"]=="http://localhost:8080/oidc/token"'

if docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app -d app -c 'CREATE TABLE forbidden_runtime_schema_change(id text)'; then
  echo "runtime role changed schema" >&2; exit 1
fi
if docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app -d app -c 'UPDATE schema_migrations SET dirty=true'; then
  echo "runtime role mutated migration ledger" >&2; exit 1
fi
if docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U app_migrator -d app -c 'TRUNCATE audit_event_models'; then
  echo "database owner bypassed append-only audit guard" >&2; exit 1
fi
docker compose exec -T postgres psql -At -U app -d app -v "owner_id=$owner_id" <<'AUDIT_SQL' | grep -Eq '^[1-9][0-9]*$'
SELECT count(*) FROM audit_event_models WHERE owner_user_id=:'owner_id' AND action='user.viewed';
AUDIT_SQL

docker compose stop spicedb
expect_status 503 http://localhost:8080/readyz
expect_status 204 http://localhost:8080/livez
docker compose start spicedb
for _ in $(seq 1 60); do curl -fsS http://localhost:8080/readyz >/dev/null 2>&1 && break; sleep 1; done
echo "blank-app authentication, OIDC, audit, role, and health acceptance passed"
