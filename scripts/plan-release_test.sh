#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"; tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q; git -C "$tmp" config user.email test@example.invalid; git -C "$tmp" config user.name Test
touch "$tmp/file"; git -C "$tmp" add file; git -C "$tmp" commit -qm 'feat: initial capability'
result="$(cd "$tmp" && "$root/scripts/plan-release.sh")"; grep -qx 'next_tag=v0.1.0' <<<"$result"
git -C "$tmp" tag v0.1.0; echo change >> "$tmp/file"; git -C "$tmp" commit -qam 'fix: correction'
result="$(cd "$tmp" && "$root/scripts/plan-release.sh")"; grep -qx 'next_tag=v0.1.1' <<<"$result"
