#!/usr/bin/env bash
set -euo pipefail

write_output=false
if [ "${1:-}" = "--github-output" ]; then write_output=true; elif [ "$#" -gt 0 ]; then echo "usage: $0 [--github-output]" >&2; exit 2; fi
latest="$(git tag --merged HEAD --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | head -n1)"
if [ -n "$latest" ]; then range="$latest..HEAD"; previous="$latest"; version="${latest#v}"; else range=HEAD; previous=none; version=0.0.0; fi
subjects="$(git log --format='%s' "$range")"
bodies="$(git log --format='%b' "$range")"
bump=none
if printf '%s\n%s\n' "$subjects" "$bodies" | grep -Eq '(^|[[:space:]])BREAKING CHANGE:|^[a-zA-Z]+(\([^)]+\))?!:'; then bump=major
elif printf '%s\n' "$subjects" | grep -Eq '^feat(\([^)]+\))?:'; then bump=minor
elif printf '%s\n' "$subjects" | grep -Eq '^(fix|perf)(\([^)]+\))?:'; then bump=patch; fi
IFS=. read -r major minor patch <<EOF
$version
EOF
case "$bump" in major) major=$((major+1)); minor=0; patch=0;; minor) minor=$((minor+1)); patch=0;; patch) patch=$((patch+1));; esac
next="$major.$minor.$patch"; required=false; [ "$bump" = none ] || required=true
emit(){ printf '%s=%s\n' "$1" "$2"; if [ "$write_output" = true ]; then : "${GITHUB_OUTPUT:?required}"; printf '%s=%s\n' "$1" "$2" >> "$GITHUB_OUTPUT"; fi; }
emit release_required "$required"; emit previous_tag "$previous"; emit next_version "$next"; emit next_tag "v$next"; emit bump "$bump"
