#!/usr/bin/env bash
set -euo pipefail

readonly build_log="${1:?usage: $0 BUILD_LOG STEP_SUBSTRING}"
readonly step_substring="${2:?usage: $0 BUILD_LOG STEP_SUBSTRING}"

if [[ ! -s "$build_log" ]]; then
  echo "BuildKit log is missing or empty: $build_log" >&2
  exit 1
fi

if ! awk -v step="$step_substring" '
  index($0, step) { stage[$1] = 1; observed = 1 }
  stage[$1] && $0 ~ /(^|[[:space:]])CACHED([[:space:]]|$)/ { hit = 1 }
  END { exit !(observed && hit) }
' "$build_log"; then
  echo "required BuildKit cache hit was not observed for step: $step_substring" >&2
  exit 1
fi

echo "Observed BuildKit cache hit for: $step_substring"
