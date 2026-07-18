#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"
if [[ ! -f .env ]]; then echo "missing .env; copy .env.example to .env" >&2; exit 1; fi
set -a
# shellcheck disable=SC1091
source ./.env
set +a
cd apps/api
exec go run ./cmd/server
