#!/usr/bin/env bash
set -euo pipefail

readonly root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly subject="$root/scripts/publish-release.sh"
readonly source_sha="0123456789abcdef0123456789abcdef01234567"
readonly tag="v9.8.7"
readonly test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT

grep -q 'RELEASE_UPLOAD_ATTEMPTS:-5' "$subject"
grep -q 'gh release upload.*--clobber' "$subject"
grep -q 'gh release edit.*--draft=false' "$subject"

mkdir -p "$test_dir/bin" "$test_dir/dist"
for asset in make-app_linux_amd64.tar.gz make-app_linux_arm64.tar.gz make-app_darwin_amd64.tar.gz make-app_darwin_arm64.tar.gz checksums.txt; do
  printf 'fixture\n' >"$test_dir/dist/$asset"
done

cat >"$test_dir/bin/git" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  "rev-parse HEAD"|"rev-parse ${RELEASE_TEST_TAG}^{commit}") printf '%s\n' "$RELEASE_TEST_SOURCE" ;;
  "ls-remote --tags origin refs/tags/${RELEASE_TEST_TAG}") printf '%s\trefs/tags/%s\n' "$RELEASE_TEST_SOURCE" "$RELEASE_TEST_TAG" ;;
  *) echo "unexpected git call: $*" >&2; exit 64 ;;
esac
FAKE
cat >"$test_dir/bin/gh" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$RELEASE_TEST_CALLS"
case "$*" in
  "release view ${RELEASE_TEST_TAG} --json isDraft --jq .isDraft") printf 'true\n' ;;
  "release upload ${RELEASE_TEST_TAG} "*" --clobber")
    attempts=0
    [[ -f "$RELEASE_TEST_ATTEMPTS" ]] && attempts="$(cat "$RELEASE_TEST_ATTEMPTS")"
    attempts=$((attempts + 1))
    printf '%s\n' "$attempts" >"$RELEASE_TEST_ATTEMPTS"
    ((attempts >= 3))
    ;;
  "release edit ${RELEASE_TEST_TAG} --draft=false") ;;
  *) echo "unexpected gh call: $*" >&2; exit 64 ;;
esac
FAKE
chmod +x "$test_dir/bin/git" "$test_dir/bin/gh"

RELEASE_TEST_SOURCE="$source_sha" RELEASE_TEST_TAG="$tag" \
RELEASE_TEST_CALLS="$test_dir/gh.calls" RELEASE_TEST_ATTEMPTS="$test_dir/attempts" \
RELEASE_RETRY_DELAY_SECONDS=0 PATH="$test_dir/bin:$PATH" \
  "$subject" "$source_sha" "$tag" "$test_dir/dist"

test "$(grep -c "^release upload $tag " "$test_dir/gh.calls")" -eq 7
grep -qx "release edit $tag --draft=false" "$test_dir/gh.calls"

if RELEASE_TEST_SOURCE="$source_sha" RELEASE_TEST_TAG="$tag" \
  RELEASE_TEST_CALLS="$test_dir/mismatch.calls" PATH="$test_dir/bin:$PATH" \
  "$subject" fedcba9876543210fedcba9876543210fedcba98 "$tag" "$test_dir/dist" >"$test_dir/mismatch.out" 2>&1; then
  echo "publication accepted a source other than checked-out HEAD" >&2
  exit 1
fi
test ! -e "$test_dir/mismatch.calls"

grep -q 'scripts/publish-release.sh "$SOURCE_SHA"' "$root/.github/workflows/release.yml"
echo "release publication is immutable, restart-safe, and retry-bounded"
