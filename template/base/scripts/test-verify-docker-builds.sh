#!/usr/bin/env bash
set -euo pipefail

readonly root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly subject="$root/scripts/verify-docker-builds.sh"
readonly cache_subject="$root/scripts/assert-buildkit-cache-hit.sh"
readonly test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT

cat >"$test_dir/cache-hit.log" <<'LOG'
#22 [build 6/6] RUN CGO_ENABLED=0 go build -o /out/ ./cmd/server
#22 CACHED
LOG
"$cache_subject" "$test_dir/cache-hit.log" 'CGO_ENABLED=0 go build'
sed 's/CACHED/DONE 42.1s/' "$test_dir/cache-hit.log" >"$test_dir/cache-miss.log"
if "$cache_subject" "$test_dir/cache-miss.log" 'CGO_ENABLED=0 go build' >"$test_dir/cache-miss.out" 2>&1; then
  echo "an expensive uncached build step must fail verification" >&2
  exit 1
fi

mkdir -p "$test_dir/bin" "$test_dir/logs"
cat >"$test_dir/bin/docker" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$DOCKER_TEST_CALLS"
case "$*" in
  "buildx version") printf '%s\n' 'github.com/docker/buildx v0.35.0 fixture' ;;
  "buildx inspect default --bootstrap"|"compose build api web"|"image inspect "*-api:verification" "*-web:verification) ;;
  "buildx build --builder default --load --progress plain --file apps/api/Dockerfile"*)
    printf '%s\n' '#41 [build 7/7] RUN CGO_ENABLED=0 go build -o /out/ ./cmd/server' '#41 CACHED'
    ;;
  "buildx build --builder default --load --progress plain --file apps/web/Dockerfile"*)
    printf '%s\n' '#52 [build 8/8] RUN pnpm --dir apps/web build' '#52 CACHED'
    ;;
  *) echo "unexpected docker call: $*" >&2; exit 64 ;;
esac
FAKE
cat >"$test_dir/bin/gh" <<'FAKE'
#!/usr/bin/env bash
echo "gh must not run with the pinned Buildx fixture" >&2
exit 99
FAKE
chmod +x "$test_dir/bin/docker" "$test_dir/bin/gh"

DOCKER_TEST_CALLS="$test_dir/docker.calls" TMPDIR="$test_dir/logs" PATH="$test_dir/bin:$PATH" "$subject"
grep -qx 'compose build api web' "$test_dir/docker.calls"
test "$(grep -c '^buildx build --builder default --load --progress plain' "$test_dir/docker.calls")" -eq 2
grep -q '^image inspect .*api:verification .*web:verification$' "$test_dir/docker.calls"
test -z "$(find "$test_dir/logs" -mindepth 1 -print -quit)"
if grep -Eq '^build ' "$test_dir/docker.calls"; then
  echo "verification fell back to the legacy builder" >&2
  exit 1
fi

echo "Docker verification shares BuildKit cache and cleans its evidence logs"
