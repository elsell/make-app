#!/usr/bin/env bash
set -euo pipefail

fail=0
report() { printf 'structural check: %s\n' "$*" >&2; fail=1; }

for required in \
  apps/api/internal/domain/audit/event.go \
  apps/api/internal/adapters/dbmigrations/000004_create_audit_events.up.sql \
  specs/audit/audit.spec.md; do
  [[ -f "$required" ]] || report "mandatory audit primitive is missing $required"
done
if [[ -f apps/api/internal/adapters/dbmigrations/000004_create_audit_events.up.sql ]] &&
  ! grep -q 'BEFORE UPDATE OR DELETE ON audit_event_models' apps/api/internal/adapters/dbmigrations/000004_create_audit_events.up.sql; then
  report "audit persistence is not database-enforced append-only"
fi

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

grype_installer="scripts/install-grype.sh"
release_workflow=".github/workflows/release.yml"
if [[ ! -x "$grype_installer" ]]; then
  report "checksum-verified Grype installer is missing or not executable"
else
  for required in \
    'grype_0.111.1_linux_amd64.tar.gz' \
    '2bc0bc60f1f4e10b0429f5e84517ec4cf0d769d2ef66875c64fc6640e136fd8f' \
    'https://github.com/anchore/grype/releases/download/v0.111.1/grype_0.111.1_linux_amd64.tar.gz' \
    'sha256sum --check' \
    './.bin/grype version'; do
    grep -Fq "$required" "$grype_installer" || report "Grype installer lost reviewed pin: $required"
  done
fi

if [[ ! -f "$release_workflow" ]]; then
  report "release workflow is missing"
else
  grep -Fq 'run: ./scripts/install-grype.sh' "$release_workflow" || report "release workflow does not install verified Grype"
  grep -Fq 'run: ./.bin/grype "${{ steps.images.outputs.api }}" --fail-on high' "$release_workflow" || report "release workflow does not scan the API image with local Grype"
  grep -Fq 'run: ./.bin/grype "${{ steps.images.outputs.web }}" --fail-on high' "$release_workflow" || report "release workflow does not scan the web image with local Grype"
  install_line="$(awk 'index($0, "run: ./scripts/install-grype.sh") { print NR; exit }' "$release_workflow")"
  api_scan_line="$(awk 'index($0, "run: ./.bin/grype \"${{ steps.images.outputs.api }}\"") { print NR; exit }' "$release_workflow")"
  web_scan_line="$(awk 'index($0, "run: ./.bin/grype \"${{ steps.images.outputs.web }}\"") { print NR; exit }' "$release_workflow")"
  publish_line="$(awk 'index($0, "id: publish") { print NR; exit }' "$release_workflow")"
  if [[ -z "$install_line" || -z "$api_scan_line" || -z "$web_scan_line" || -z "$publish_line" ]] ||
    (( install_line >= api_scan_line || install_line >= web_scan_line || api_scan_line >= publish_line || web_scan_line >= publish_line )); then
    report "verified Grype installation and scans must precede image publication"
  fi
fi

if grep -RInE 'anchore/scan-action@|raw\.githubusercontent\.com/anchore/grype/[^/]+/install\.sh|https?://get\.anchore\.io/grype|(curl|wget)[^|]*\|[[:space:]]*(ba)?sh' "$release_workflow" "$grype_installer" >/dev/null 2>&1; then
  report "Grype scans must not execute a remote installer"
fi

while IFS= read -r line; do
  [[ "$line" == *'@sha256:'* ]] || report "Compose image is not digest-pinned: $line"
done < <(grep -hE '^[[:space:]]+image:[[:space:]]+' compose*.yaml 2>/dev/null || true)

while IFS= read -r line; do
  [[ "$line" == *'@sha256:'* ]] || report "Dockerfile base is not digest-pinned: $line"
done < <(grep -hE '^FROM[[:space:]]+' apps/*/Dockerfile 2>/dev/null || true)

exit "$fail"
