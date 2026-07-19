#!/usr/bin/env bash
set -euo pipefail

root="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
manifest="$root/.release-version"
[[ -f "$manifest" ]] || { echo "release docs: missing .release-version" >&2; exit 1; }
version="$(tr -d '\r\n' < "$manifest")"
[[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "release docs: invalid manifest version" >&2; exit 1; }

check_marked_versions() {
  local file="$1" start='<!-- make-app:release-version:start -->' end='<!-- make-app:release-version:end -->'
  [[ "$(grep -Fo "$start" "$file" | wc -l | tr -d ' ')" -eq 1 ]] || { echo "release docs: $file must contain one start marker" >&2; exit 1; }
  [[ "$(grep -Fo "$end" "$file" | wc -l | tr -d ' ')" -eq 1 ]] || { echo "release docs: $file must contain one end marker" >&2; exit 1; }
  local versions
  versions="$(awk -v start="$start" -v end="$end" 'index($0, start) { marked=1 } marked { print } index($0, end) { marked=0 }' "$file" | grep -Eo 'v[0-9]+\.[0-9]+\.[0-9]+' | sort -u || true)"
  [[ "$versions" == "$version" ]] || { echo "release docs: $file marked version is stale (expected $version)" >&2; exit 1; }
}

check_marked_versions "$root/README.md"
check_marked_versions "$root/docs/compatibility.md"
grep -Fqx "go install github.com/elsell/make-app@$version" "$root/README.md" || { echo "release docs: README install command is stale" >&2; exit 1; }
if grep -Fq 'github.com/elsell/make-app@latest' "$root/README.md"; then
  echo "release docs: README install command must not float at @latest" >&2
  exit 1
fi
echo "release documentation matches $version"
