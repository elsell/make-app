#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
cd "$root"
go build -o "$work/make-app" .
"$work/make-app" new 'Feature Acceptance' --module example.com/features --output "$work/app" --without-example
for feature in notifications media testflight selfhost; do "$work/make-app" feature add "$feature" --dir "$work/app"; done
cd "$work/app"
pnpm install --frozen-lockfile
make generate
(cd apps/api && GOWORK=off go mod tidy)
node scripts/check-app-env.mjs
node scripts/check-client-api-boundary.mjs
node scripts/check-client-api-boundary.test.mjs
pnpm --dir apps/web check
pnpm --dir packages/client-core check
pnpm --dir packages/client-core test
pnpm --dir apps/mobile check
for check in scripts/check-feature-*.sh; do bash "$check"; done
go test -race ./apps/api/internal/platform/notifications/...
go run ./scripts/go-boundaries/main.go
python3 scripts/check-generated-contracts.py
python3 scripts/check-dependency-age.py
mkdir -p .bin
(cd tools && go build -o ../.bin/govulncheck golang.org/x/vuln/cmd/govulncheck)
(cd modules/media && GOWORK=off ../../.bin/govulncheck ./...)
