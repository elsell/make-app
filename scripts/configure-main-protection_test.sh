#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
script="$root/scripts/configure-main-protection.sh"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

dry_run="$($script)"
python3 -c '
import json, sys
policy = json.load(sys.stdin)
assert policy["required_status_checks"] == {
    "strict": True,
    "contexts": [
        "verify",
        "acceptance",
        "generation-matrix (ubuntu-24.04)",
        "generation-matrix (macos-15)",
        "android-native",
    ],
}
assert policy["enforce_admins"] is True
assert policy["required_pull_request_reviews"]["required_approving_review_count"] == 0
assert policy["required_conversation_resolution"] is True
assert policy["required_linear_history"] is True
assert policy["allow_force_pushes"] is False
assert policy["allow_deletions"] is False
' <<<"$dry_run"

mkdir -p "$work/bin"
cat >"$work/bin/gh" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$GH_CALLS"
case "$*" in
  "api --method PUT repos/elsell/make-app/branches/main/protection --input -")
    cat >"$GH_PUT_BODY"
    printf '{}\n'
    ;;
  "api repos/elsell/make-app/branches/main/protection")
    if [[ "${GH_BROKEN_READBACK:-}" == "1" ]]; then
      printf '%s\n' '{"required_status_checks":{"strict":true,"contexts":["verify"]}}'
    else
      cat <<'JSON'
{"required_status_checks":{"strict":true,"contexts":["verify","acceptance","generation-matrix (ubuntu-24.04)","generation-matrix (macos-15)","android-native"]},"enforce_admins":{"enabled":true},"required_pull_request_reviews":{"dismiss_stale_reviews":false,"require_code_owner_reviews":false,"required_approving_review_count":0,"require_last_push_approval":false},"required_linear_history":{"enabled":true},"allow_force_pushes":{"enabled":false},"allow_deletions":{"enabled":false},"required_conversation_resolution":{"enabled":true}}
JSON
    fi
    ;;
  *)
    echo "unexpected gh invocation: $*" >&2
    exit 90
    ;;
esac
MOCK
chmod +x "$work/bin/gh"

export GH_CALLS="$work/calls"
export GH_PUT_BODY="$work/put-body"
MAKE_APP_GH_BIN="$work/bin/gh" "$script" --apply
python3 -c 'import json, sys; policy=json.load(open(sys.argv[1])); assert len(policy["required_status_checks"]["contexts"]) == 5' "$GH_PUT_BODY"
grep -qx 'api --method PUT repos/elsell/make-app/branches/main/protection --input -' "$GH_CALLS"
grep -qx 'api repos/elsell/make-app/branches/main/protection' "$GH_CALLS"

if GH_BROKEN_READBACK=1 MAKE_APP_GH_BIN="$work/bin/gh" "$script" --check 2>"$work/broken-readback"; then
  echo "broken branch-protection readback was accepted" >&2
  exit 1
fi
grep -q 'branch protection readback mismatch' "$work/broken-readback"

echo "main branch protection configuration is dry-run-safe and readback-verified"
