#!/usr/bin/env bash
set -euo pipefail

fail=0
report() { printf 'structural check: %s\n' "$*" >&2; fail=1; }

if ! node scripts/check-i18n.mjs; then
  report "internationalization invariant failed"
fi

while IFS= read -r -d '' file; do
  first="$(head -n 1 "$file")"
  lines="$(wc -l < "$file" | tr -d ' ')"
  if (( lines > 800 )) && [[ "$first" != '// Code generated '*' DO NOT EDIT.' ]]; then
    report "$file has $lines lines (maximum 800)"
  fi
  if grep -nE 'fmt\.(Print|Printf|Println)\(' "$file" >/dev/null; then
    report "$file uses ad hoc printing"
  fi
  if grep -nE '\.(Raw|Exec)\(' "$file" >/dev/null; then
    report "$file uses direct SQL execution"
  fi
  case "$file" in
    */internal/config/*|*/cmd/*) ;;
    *) if grep -nE 'os\.(Getenv|LookupEnv)\(' "$file" >/dev/null; then report "$file reads environment outside a boundary package"; fi ;;
  esac
done < <(find apps \( -type d \( -name node_modules -o -name .svelte-kit -o -name build -o -name dist \) -prune \) -o -type f -name '*.go' -print0)

if find . \( -type d \( -name .git -o -name node_modules -o -name .svelte-kit -o -name build -o -name dist \) -prune \) -o -type f \( -iname '*mock*.go' -o -iname '*mock*.ts' -o -iname '*mock*.tsx' \) -print | grep -q .; then
  report "mock files are forbidden; use behavioral fakes"
fi

if grep -RInE '^[[:space:]]*uses:[[:space:]]+[^#[:space:]]+@(v[0-9]+|main|master)([[:space:]]|$)' .github/workflows >/dev/null 2>&1; then
  report "CI actions must use immutable commit SHAs"
fi

while IFS= read -r line; do
  [[ "$line" == *'@sha256:'* ]] || report "Compose image is not digest-pinned: $line"
done < <(grep -hE '^[[:space:]]+image:[[:space:]]+' compose*.yaml 2>/dev/null || true)

while IFS= read -r line; do
  [[ "$line" == *'@sha256:'* ]] || report "Dockerfile base is not digest-pinned: $line"
done < <(grep -hE '^FROM[[:space:]]+' apps/*/Dockerfile 2>/dev/null || true)

exit "$fail"
