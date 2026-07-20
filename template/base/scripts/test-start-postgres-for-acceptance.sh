#!/usr/bin/env bash
set -euo pipefail

readonly root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly subject="${1:-$root/scripts/start-postgres-for-acceptance.sh}"
readonly test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT

mkdir -p "$test_dir/bin"
cat >"$test_dir/bin/docker" <<'FAKE_DOCKER'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$FAKE_DOCKER_STATE/calls"
case "$*" in
  'compose pull --policy missing postgres')
    count_file="$FAKE_DOCKER_STATE/pull-count"
    count="$(( $(cat "$count_file" 2>/dev/null || printf '0') + 1 ))"
    printf '%s\n' "$count" >"$count_file"
    if (( count <= FAKE_PULL_FAILURES )); then
      echo 'short read: unexpected EOF' >&2
      exit 1
    fi
    ;;
  'compose up -d --pull never postgres')
    exit "$FAKE_UP_STATUS"
    ;;
  'compose ps -q postgres')
    printf 'postgres-container\n'
    ;;
  'inspect -f {{.State.Health.Status}} postgres-container')
    count_file="$FAKE_DOCKER_STATE/inspect-count"
    count="$(( $(cat "$count_file" 2>/dev/null || printf '0') + 1 ))"
    printf '%s\n' "$count" >"$count_file"
    if (( count >= FAKE_HEALTHY_AFTER )); then printf 'healthy\n'; else printf 'starting\n'; fi
    ;;
  *)
    echo "unexpected docker invocation: $*" >&2
    exit 97
    ;;
esac
FAKE_DOCKER
cat >"$test_dir/bin/sleep" <<'FAKE_SLEEP'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$FAKE_DOCKER_STATE/sleeps"
FAKE_SLEEP
chmod +x "$test_dir/bin/docker" "$test_dir/bin/sleep"

run_subject() {
  local state_dir="$1" pull_failures="$2" up_status="$3" healthy_after="$4"
  mkdir -p "$state_dir"
  : >"$state_dir/calls"
  : >"$state_dir/sleeps"
  FAKE_DOCKER_STATE="$state_dir" \
  FAKE_PULL_FAILURES="$pull_failures" \
  FAKE_UP_STATUS="$up_status" \
  FAKE_HEALTHY_AFTER="$healthy_after" \
  PATH="$test_dir/bin:$PATH" \
    "$subject"
}

call_count() {
  local expected="$1" calls="$2"
  awk -v expected="$expected" '$0 == expected { count++ } END { print count + 0 }' "$calls"
}

immediate="$test_dir/immediate"
run_subject "$immediate" 0 0 1
[[ "$(call_count 'compose pull --policy missing postgres' "$immediate/calls")" == 1 ]]
[[ "$(call_count 'compose up -d --pull never postgres' "$immediate/calls")" == 1 ]]
[[ ! -s "$immediate/sleeps" ]]

transient="$test_dir/transient"
run_subject "$transient" 2 0 1 >"$transient.out" 2>&1
[[ "$(call_count 'compose pull --policy missing postgres' "$transient/calls")" == 3 ]]
[[ "$(call_count 'compose up -d --pull never postgres' "$transient/calls")" == 1 ]]
[[ "$(grep -c '^2$' "$transient/sleeps")" == 2 ]]

persistent="$test_dir/persistent"
if run_subject "$persistent" 99 0 1 >"$persistent.out" 2>&1; then
  echo "persistent image acquisition failure must fail acceptance" >&2
  exit 1
fi
[[ "$(call_count 'compose pull --policy missing postgres' "$persistent/calls")" == 3 ]]
[[ "$(call_count 'compose up -d --pull never postgres' "$persistent/calls")" == 0 ]]
grep -Fq 'failed to pull PostgreSQL image after 3 attempts' "$persistent.out"

startup="$test_dir/startup"
if run_subject "$startup" 0 23 1 >"$startup.out" 2>&1; then
  echo "container startup failure must fail acceptance" >&2
  exit 1
fi
[[ "$(call_count 'compose pull --policy missing postgres' "$startup/calls")" == 1 ]]
[[ "$(call_count 'compose up -d --pull never postgres' "$startup/calls")" == 1 ]]
[[ "$(call_count 'compose ps -q postgres' "$startup/calls")" == 0 ]]

unhealthy="$test_dir/unhealthy"
if run_subject "$unhealthy" 0 0 999 >"$unhealthy.out" 2>&1; then
  echo "PostgreSQL health timeout must fail acceptance" >&2
  exit 1
fi
[[ "$(call_count 'compose pull --policy missing postgres' "$unhealthy/calls")" == 1 ]]
[[ "$(call_count 'compose up -d --pull never postgres' "$unhealthy/calls")" == 1 ]]
[[ "$(call_count 'compose ps -q postgres' "$unhealthy/calls")" == 60 ]]
grep -Fq 'PostgreSQL did not become healthy after 60 seconds' "$unhealthy.out"

echo "PostgreSQL acceptance startup retries only bounded image acquisition"
