#!/usr/bin/env bash
set -euo pipefail

readonly root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly subject="$root/scripts/install-docker-buildx.sh"
readonly test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT

grep -q 'readonly version="v0.35.0"' "$subject"
grep -q 'readonly asset="buildx-v0.35.0.linux-amd64"' "$subject"
grep -q 'readonly sha256="d41ece72044243b4f58b343441ae37446d9c29a7d6b5e11c61847bbcf8f7dfda"' "$subject"
grep -q 'sha256sum --check' "$subject"
grep -q 'gh release download' "$subject"
if grep -Eq 'curl|wget|latest' "$subject"; then
  echo "Buildx provisioning must use only its pinned GitHub release asset" >&2
  exit 1
fi

mkdir -p "$test_dir/bin"
cat >"$test_dir/bin/docker" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
test "$*" = "buildx version"
printf '%s\n' 'github.com/docker/buildx v0.35.0 fixture'
FAKE
cat >"$test_dir/bin/gh" <<'FAKE'
#!/usr/bin/env bash
echo "gh must not run when Buildx is already installed" >&2
exit 99
FAKE
chmod +x "$test_dir/bin/docker" "$test_dir/bin/gh"
PATH="$test_dir/bin:$PATH" "$subject"

cat >"$test_dir/bin/docker" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
test "$*" = "buildx version"
if [[ -f "$BUILDX_TEST_INSTALLED" ]]; then
  printf '%s\n' 'github.com/docker/buildx v0.35.0 fixture'
else
  printf '%s\n' 'github.com/docker/buildx v0.34.0 fixture'
fi
FAKE
cat >"$test_dir/bin/gh" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
test "$1 $2 $3 $4" = 'release download v0.35.0 --repo'
while (($#)); do
  case "$1" in
    --dir) shift; destination="$1" ;;
    --pattern) shift; asset="$1" ;;
  esac
  shift
done
printf '#!/usr/bin/env bash\n' >"$destination/$asset"
touch "$BUILDX_TEST_INSTALLED"
FAKE
cat >"$test_dir/bin/sha256sum" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
test "$1" = '--check' || test "$2" = '--check'
read -r checksum asset
test "$checksum" = 'd41ece72044243b4f58b343441ae37446d9c29a7d6b5e11c61847bbcf8f7dfda'
test "${asset##*/}" = 'buildx-v0.35.0.linux-amd64'
FAKE
cat >"$test_dir/bin/uname" <<'FAKE'
#!/usr/bin/env bash
case "$1" in
  -s) printf 'Linux\n' ;;
  -m) printf 'x86_64\n' ;;
  *) exit 64 ;;
esac
FAKE
chmod +x "$test_dir/bin/docker" "$test_dir/bin/gh" "$test_dir/bin/sha256sum" "$test_dir/bin/uname"
mkdir -p "$test_dir/docker-config"
BUILDX_TEST_INSTALLED="$test_dir/installed" \
DOCKER_CONFIG="$test_dir/docker-config" PATH="$test_dir/bin:$PATH" \
  "$subject"
test -x "$test_dir/docker-config/cli-plugins/docker-buildx"
test -f "$test_dir/installed"

echo "Buildx provisioning is immutable and idempotent"
