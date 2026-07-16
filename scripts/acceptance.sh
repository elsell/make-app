#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work="$(mktemp -d)"
cleanup() {
  if [[ -f "$work/secure-app/compose.yaml" ]]; then
    docker compose -f "$work/secure-app/compose.yaml" down --volumes --remove-orphans >/dev/null 2>&1 || true
  fi
  rm -rf "$work"
}
trap cleanup EXIT

cd "$root"
go test -race ./...
go run . new "Secure App" --module example.com/secure-app --output "$work/secure-app"

cd "$work/secure-app"
test -x .git/hooks/pre-commit
make bootstrap
make check
git add .
git diff --cached --exit-code -- packages/api-client/openapi.json packages/api-client/src/schema.d.ts
pnpm build
docker compose config -q

echo "static generated-project acceptance passed"
