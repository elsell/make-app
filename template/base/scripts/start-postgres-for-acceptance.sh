#!/usr/bin/env bash
set -euo pipefail

readonly pull_attempt_limit=3
pull_attempt=1
while ! docker compose pull --policy missing postgres; do
  if (( pull_attempt >= pull_attempt_limit )); then
    echo "failed to pull PostgreSQL image after $pull_attempt_limit attempts" >&2
    exit 1
  fi
  echo "PostgreSQL image pull attempt $pull_attempt failed; retrying in 2 seconds" >&2
  sleep 2
  pull_attempt="$((pull_attempt + 1))"
done

# Image acquisition is complete. A startup/configuration failure is not a
# registry-transfer failure and must fail immediately without retry.
docker compose up -d --pull never postgres

for health_attempt in $(seq 1 60); do
  id="$(docker compose ps -q postgres)"
  if [[ -n "$id" ]] && [[ "$(docker inspect -f '{{.State.Health.Status}}' "$id")" == healthy ]]; then
    exit 0
  fi
  if (( health_attempt < 60 )); then sleep 1; fi
done

echo "PostgreSQL did not become healthy after 60 seconds" >&2
exit 1
