#!/usr/bin/env bash
set -euo pipefail

readonly root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly build_log_dir="$(mktemp -d "${TMPDIR:-/tmp}/__APP_SLUG__-build-logs.XXXXXX")"
trap 'rm -rf "$build_log_dir"' EXIT

cd "$root"
./scripts/install-docker-buildx.sh
export BUILDX_BUILDER="${BUILDX_BUILDER:-default}"
export COMPOSE_BAKE=true
./scripts/require-docker-buildkit.sh

# Populate the selected builder's cache through the same Compose path used by
# local development and acceptance, then require standalone release-style builds
# to reuse the expensive application layers.
docker compose build api web

readonly api_build_log="$build_log_dir/api-build.log"
readonly web_build_log="$build_log_dir/web-build.log"
docker buildx build --builder "$BUILDX_BUILDER" --load --progress plain --file apps/api/Dockerfile --tag __APP_SLUG__-api:verification . 2>&1 | tee "$api_build_log"
docker buildx build --builder "$BUILDX_BUILDER" --load --progress plain --file apps/web/Dockerfile --tag __APP_SLUG__-web:verification . 2>&1 | tee "$web_build_log"
./scripts/assert-buildkit-cache-hit.sh "$api_build_log" 'CGO_ENABLED=0 go build'
./scripts/assert-buildkit-cache-hit.sh "$web_build_log" 'pnpm --dir apps/web build'
docker image inspect __APP_SLUG__-api:verification __APP_SLUG__-web:verification >/dev/null

echo "BuildKit cache reuse and loaded production images verified"
