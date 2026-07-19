#!/usr/bin/env bash
set -euo pipefail

readonly root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly subject="$root/scripts/require-docker-buildkit.sh"
readonly test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT

mkdir -p "$test_dir/bin"
cat >"$test_dir/bin/docker" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$BUILDKIT_TEST_CALLS"
case "$*" in
  "buildx version")
    printf '%s\n' "github.com/docker/buildx ${BUILDKIT_TEST_VERSION:-v0.35.0} fixture"
    exit "${BUILDKIT_TEST_VERSION_STATUS:-0}"
    ;;
  "buildx inspect ${BUILDX_BUILDER:-default} --bootstrap") exit "${BUILDKIT_TEST_INSPECT_STATUS:-0}" ;;
  *) exit 64 ;;
esac
FAKE
chmod +x "$test_dir/bin/docker"

BUILDKIT_TEST_CALLS="$test_dir/success.calls" PATH="$test_dir/bin:$PATH" "$subject"
grep -qx 'buildx inspect default --bootstrap' "$test_dir/success.calls"

if BUILDKIT_TEST_CALLS="$test_dir/version.calls" BUILDKIT_TEST_VERSION=v0.34.0 PATH="$test_dir/bin:$PATH" "$subject" >"$test_dir/version.out" 2>&1; then
  echo "a non-pinned Buildx version must fail closed" >&2
  exit 1
fi
grep -q 'requires docker buildx v0.35.0' "$test_dir/version.out"
if grep -q inspect "$test_dir/version.calls"; then
  echo "builder inspection must not run with the wrong Buildx version" >&2
  exit 1
fi

if BUILDKIT_TEST_CALLS="$test_dir/missing.calls" BUILDKIT_TEST_VERSION_STATUS=1 PATH="$test_dir/bin:$PATH" "$subject" >"$test_dir/missing.out" 2>&1; then
  echo "missing Buildx must fail closed" >&2
  exit 1
fi
grep -q 'docker buildx is unavailable' "$test_dir/missing.out"
if grep -q inspect "$test_dir/missing.calls"; then
  echo "builder inspection must not run without Buildx" >&2
  exit 1
fi

if BUILDKIT_TEST_CALLS="$test_dir/inspect.calls" BUILDKIT_TEST_INSPECT_STATUS=1 PATH="$test_dir/bin:$PATH" "$subject" >"$test_dir/inspect.out" 2>&1; then
  echo "an unusable builder must fail closed" >&2
  exit 1
fi
grep -q "selected BuildKit builder 'default' is unavailable" "$test_dir/inspect.out"

echo "BuildKit capability verification fails closed"
