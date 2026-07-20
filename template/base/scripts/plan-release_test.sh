#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
caller_root="$(git rev-parse --show-toplevel)"
caller_git_dir="$(git rev-parse --absolute-git-dir)"
caller_index="$(git rev-parse --git-path index)"
case "$caller_index" in /*) ;; *) caller_index="$PWD/$caller_index" ;; esac
caller_staged_before="$(git diff --cached --name-status)"
git_local_env_vars="$(git rev-parse --local-env-vars)"
while IFS= read -r variable; do
  [[ -z "$variable" ]] || unset "$variable"
done <<<"$git_local_env_vars"

tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" config core.hooksPath /dev/null
git -C "$tmp" config user.email test@example.invalid
git -C "$tmp" config user.name Test
touch "$tmp/file"; git -C "$tmp" add file; git -C "$tmp" commit -qm 'feat: initial capability'
result="$(cd "$tmp" && "$root/scripts/plan-release.sh")"
grep -qx 'next_tag=v0.1.0' <<<"$result"
git -C "$tmp" tag v0.1.0; echo change >> "$tmp/file"; git -C "$tmp" commit -qam 'fix: correct behavior'
result="$(cd "$tmp" && "$root/scripts/plan-release.sh")"
grep -qx 'next_tag=v0.1.1' <<<"$result"
caller_staged_after="$(GIT_DIR="$caller_git_dir" GIT_WORK_TREE="$caller_root" GIT_INDEX_FILE="$caller_index" git diff --cached --name-status)"
[[ "$caller_staged_after" == "$caller_staged_before" ]] || { echo "release-plan test changed the caller index" >&2; exit 1; }
