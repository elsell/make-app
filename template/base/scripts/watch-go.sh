#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fingerprint() {
  python3 - "$root/apps/api" <<'PY'
import hashlib, os, sys
digest = hashlib.sha256()
for base, _, names in os.walk(sys.argv[1]):
    for name in sorted(names):
        if not name.endswith((".go", ".sql")):
            continue
        path = os.path.join(base, name)
        digest.update(os.path.relpath(path, sys.argv[1]).encode())
        with open(path, "rb") as source:
            digest.update(source.read())
print(digest.hexdigest())
PY
}
child=""
cleanup() { if [[ -n "$child" ]]; then kill "$child" 2>/dev/null || true; wait "$child" 2>/dev/null || true; fi; }
trap cleanup EXIT INT TERM
previous=""
while true; do
  current="$(fingerprint)"
  if [[ "$current" != "$previous" ]]; then
    cleanup
    "$root/scripts/run-api.sh" & child=$!
    previous="$current"
  elif [[ -n "$child" ]] && ! kill -0 "$child" 2>/dev/null; then
    wait "$child" || true
    child=""
  fi
  sleep 1
done
