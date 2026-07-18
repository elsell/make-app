#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"
if [[ ! -f .env ]]; then echo "missing .env; copy .env.example to .env" >&2; exit 1; fi
docker compose up -d postgres dex spicedb-runtime-proxy
docker compose run --rm app-migrate
./scripts/watch-go.sh & api=$!
pnpm --dir apps/web dev --host 127.0.0.1 & web=$!
cleanup() { kill "$api" "$web" 2>/dev/null || true; wait "$api" "$web" 2>/dev/null || true; }
trap cleanup EXIT INT TERM
echo "Development ready: web http://localhost:5173 | API http://localhost:8080"
while kill -0 "$api" 2>/dev/null && kill -0 "$web" 2>/dev/null; do sleep 1; done
exit 1
