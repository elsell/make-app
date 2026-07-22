#!/usr/bin/env bash
set -euo pipefail
workspace="$(find apps/mobile/ios -maxdepth 1 -name '*.xcworkspace' -print -quit)"
test -n "$workspace" || { echo "iOS workspace missing; run make mobile-prebuild" >&2; exit 1; }
application_scheme="$(basename "$workspace" .xcworkspace)"
schemes_json="$(xcodebuild -list -json -workspace "$workspace")"
scheme="$(printf '%s' "$schemes_json" | python3 -c 'import json,sys; target=sys.argv[1]; schemes=json.load(sys.stdin).get("workspace",{}).get("schemes",[]); sys.stdout.write(target if target in schemes else "")' "$application_scheme")"
test -n "$scheme" || { echo "iOS application scheme \"$application_scheme\" missing" >&2; exit 1; }
xcodebuild -workspace "$workspace" -scheme "$scheme" -configuration Debug -sdk iphonesimulator CODE_SIGNING_ALLOWED=NO build
