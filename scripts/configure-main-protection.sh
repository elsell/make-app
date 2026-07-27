#!/usr/bin/env bash
set -euo pipefail

repository="${MAKE_APP_GITHUB_REPOSITORY:-elsell/make-app}"
endpoint="repos/$repository/branches/main/protection"
mode="dry-run"
gh_bin="${MAKE_APP_GH_BIN:-gh}"

if (($# > 1)); then
  echo "usage: $0 [--apply|--check]" >&2
  exit 2
fi
if (($# == 1)); then
  case "$1" in
    --apply) mode="apply" ;;
    --check) mode="check" ;;
    *) echo "usage: $0 [--apply|--check]" >&2; exit 2 ;;
  esac
fi

policy() {
  cat <<'JSON'
{
  "required_status_checks": {
    "strict": true,
    "contexts": [
      "verify",
      "acceptance",
      "generation-matrix (ubuntu-24.04)",
      "generation-matrix (macos-15)",
      "android-native"
    ]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": false,
    "require_code_owner_reviews": false,
    "required_approving_review_count": 0,
    "require_last_push_approval": false
  },
  "restrictions": null,
  "required_linear_history": true,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "block_creations": false,
  "required_conversation_resolution": true,
  "lock_branch": false,
  "allow_fork_syncing": false
}
JSON
}

require_gh() {
  command -v "$gh_bin" >/dev/null || { echo "$gh_bin is required to inspect or apply branch protection" >&2; exit 1; }
}

verify_readback() {
  "$gh_bin" api "$endpoint" | python3 -c '
import json, sys

actual = json.load(sys.stdin)
contexts = [
    "verify",
    "acceptance",
    "generation-matrix (ubuntu-24.04)",
    "generation-matrix (macos-15)",
    "android-native",
]
expected = {
    "required_status_checks.strict": True,
    "required_status_checks.contexts": contexts,
    "enforce_admins.enabled": True,
    "required_pull_request_reviews.dismiss_stale_reviews": False,
    "required_pull_request_reviews.require_code_owner_reviews": False,
    "required_pull_request_reviews.required_approving_review_count": 0,
    "required_pull_request_reviews.require_last_push_approval": False,
    "required_linear_history.enabled": True,
    "allow_force_pushes.enabled": False,
    "allow_deletions.enabled": False,
    "required_conversation_resolution.enabled": True,
}

def value_at(path):
    value = actual
    for part in path.split("."):
        value = value[part]
    return value

failures = []
for path, wanted in expected.items():
    try:
        found = value_at(path)
    except (KeyError, TypeError):
        failures.append(f"{path}: missing")
        continue
    if found != wanted:
        failures.append(f"{path}: expected {wanted!r}, found {found!r}")
if failures:
    print("branch protection readback mismatch:", file=sys.stderr)
    print("\n".join(f"- {failure}" for failure in failures), file=sys.stderr)
    raise SystemExit(1)
print("main branch protection readback verified")
'
}

case "$mode" in
  dry-run)
    policy
    ;;
  check)
    require_gh
    verify_readback
    ;;
  apply)
    require_gh
    policy | "$gh_bin" api --method PUT "$endpoint" --input - >/dev/null
    verify_readback
    ;;
esac
