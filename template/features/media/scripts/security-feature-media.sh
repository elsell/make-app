#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../modules/media"
GOWORK=off ../../.bin/govulncheck ./...
