#!/usr/bin/env bash
set -euo pipefail

readonly required_version="v0.35.0"
readonly builder="${BUILDX_BUILDER:-default}"

if ! version_output="$(docker buildx version 2>/dev/null)"; then
  echo "docker buildx is unavailable; run ./scripts/install-docker-buildx.sh on the Docker host" >&2
  exit 1
fi

installed_version="$(awk '{ for (field = 1; field <= NF; field++) if ($field ~ /^v[0-9]+\.[0-9]+\.[0-9]+$/) { print $field; exit } }' <<<"$version_output")"
if [[ "$installed_version" != "$required_version" ]]; then
  echo "verification requires docker buildx $required_version; found ${installed_version:-an unrecognized version}" >&2
  echo "run ./scripts/install-docker-buildx.sh on the Docker host" >&2
  exit 1
fi

if ! docker buildx inspect "$builder" --bootstrap >/dev/null 2>&1; then
  echo "the selected BuildKit builder '$builder' is unavailable; repair Docker before verification" >&2
  exit 1
fi

echo "BuildKit builder '$builder' is available with buildx $required_version"
