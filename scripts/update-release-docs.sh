#!/usr/bin/env bash
set -euo pipefail

version="${1:-}"
root="${2:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
[[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "usage: $0 vMAJOR.MINOR.PATCH [repository]" >&2; exit 2; }

update_marked_versions() {
  local file="$1" tmp start='<!-- make-app:release-version:start -->' end='<!-- make-app:release-version:end -->'
  [[ "$(grep -Fo "$start" "$file" | wc -l | tr -d ' ')" -eq 1 && "$(grep -Fo "$end" "$file" | wc -l | tr -d ' ')" -eq 1 ]] || {
    echo "release docs: refusing to update malformed markers in $file" >&2
    exit 1
  }
  tmp="$(mktemp "${file}.XXXXXX")"
  awk -v version="$version" -v start="$start" -v end="$end" '
    index($0, start) { marked=1 }
    marked { gsub(/v[0-9]+\.[0-9]+\.[0-9]+/, version) }
    { print }
    index($0, end) { marked=0 }
  ' "$file" > "$tmp"
  chmod 0644 "$tmp"
  mv "$tmp" "$file"
}

update_marked_versions "$root/README.md"
update_marked_versions "$root/docs/compatibility.md"
printf '%s\n' "$version" > "$root/.release-version"
"$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/check-release-docs.sh" "$root"
