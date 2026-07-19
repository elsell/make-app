#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/docs"
cp "$root/.release-version" "$root/README.md" "$fixture/"
cp "$root/docs/compatibility.md" "$fixture/docs/"
current="$(cat "$fixture/.release-version")"

replace_once() {
  local from="$1" to="$2" file="$3" tmp="$3.tmp"
  awk -v from="$from" -v to="$to" '!done && index($0, from) { sub(from, to); done=1 } { print }' "$file" > "$tmp"
  mv "$tmp" "$file"
}

"$root/scripts/check-release-docs.sh" "$fixture" >/dev/null

# stale README must fail the same gate used by CI.
replace_once "$current" v0.0.1 "$fixture/README.md"
if "$root/scripts/check-release-docs.sh" "$fixture" >/dev/null 2>&1; then echo "stale README passed" >&2; exit 1; fi
"$root/scripts/update-release-docs.sh" "$current" "$fixture" >/dev/null

# stale compatibility metadata must fail independently.
replace_once "$current" v0.0.1 "$fixture/docs/compatibility.md"
if "$root/scripts/check-release-docs.sh" "$fixture" >/dev/null 2>&1; then echo "stale compatibility passed" >&2; exit 1; fi
"$root/scripts/update-release-docs.sh" "$current" "$fixture" >/dev/null

# A floating @latest install must remain forbidden even inside valid markers.
replace_once "@$current" @latest "$fixture/README.md"
if "$root/scripts/check-release-docs.sh" "$fixture" >/dev/null 2>&1; then echo "@latest passed" >&2; exit 1; fi
cp "$root/README.md" "$fixture/README.md"

if "$root/scripts/update-release-docs.sh" invalid "$fixture" >/dev/null 2>&1; then echo "invalid release passed" >&2; exit 1; fi
"$root/scripts/update-release-docs.sh" v9.8.7 "$fixture" >/dev/null
grep -Fqx v9.8.7 "$fixture/.release-version"
grep -Fq 'make-app@v9.8.7' "$fixture/README.md"
grep -Fq '`v9.8.7`' "$fixture/docs/compatibility.md"
"$root/scripts/check-release-docs.sh" "$fixture" >/dev/null
echo "release documentation tests passed"
