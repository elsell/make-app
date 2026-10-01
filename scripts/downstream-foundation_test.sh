#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
cd "$root"
go run . new 'Audit App' --module example.com/audit --output "$work/app" --without-example
cd "$work/app"
node --test scripts/release-notes.test.mjs scripts/check-app-env.test.mjs scripts/validate-mobile-release-env.test.mjs
node scripts/check-app-env.mjs
bash scripts/test-hosted-android-toolchain.sh
python3 scripts/test_ci_changes.py

cd "$root"
go run . new 'Example Check' --module example.com/example-check --output "$work/example"
(cd "$work/example" && node scripts/check-app-env.mjs)
