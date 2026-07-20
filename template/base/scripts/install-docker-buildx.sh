#!/usr/bin/env bash
set -euo pipefail

readonly version="v0.35.0"
readonly asset="buildx-v0.35.0.linux-amd64"
readonly sha256="d41ece72044243b4f58b343441ae37446d9c29a7d6b5e11c61847bbcf8f7dfda"

installed_version=""
if version_output="$(docker buildx version 2>/dev/null)"; then
  installed_version="$(awk '{ for (field = 1; field <= NF; field++) if ($field ~ /^v[0-9]+\.[0-9]+\.[0-9]+$/) { print $field; exit } }' <<<"$version_output")"
  if [[ "$installed_version" = "$version" ]]; then
    echo "docker buildx $version is already installed"
    exit 0
  fi
  echo "replacing docker buildx ${installed_version:-with an unrecognized version} with $version"
fi

if [[ "$(uname -s)" != "Linux" || "$(uname -m)" != "x86_64" ]]; then
  echo "pinned Buildx provisioning supports only a Linux x86_64 Docker host" >&2
  exit 1
fi

readonly temporary_directory="$(mktemp -d)"
trap 'rm -rf "$temporary_directory"' EXIT

gh release download "$version" \
  --repo docker/buildx \
  --pattern "$asset" \
  --dir "$temporary_directory"
printf '%s  %s\n' "$sha256" "$temporary_directory/$asset" | sha256sum --check -

readonly plugin_directory="${DOCKER_CONFIG:-${HOME:?}/.docker}/cli-plugins"
mkdir -p "$plugin_directory"
install -m 0755 "$temporary_directory/$asset" "$plugin_directory/docker-buildx"
installed_output="$(docker buildx version)"
installed_version="$(awk '{ for (field = 1; field <= NF; field++) if ($field ~ /^v[0-9]+\.[0-9]+\.[0-9]+$/) { print $field; exit } }' <<<"$installed_output")"
if [[ "$installed_version" != "$version" ]]; then
  echo "installed docker buildx did not report required version $version" >&2
  exit 1
fi
printf '%s\n' "$installed_output"
