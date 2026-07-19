#!/usr/bin/env bash
set -euo pipefail
workspace="$(find apps/mobile/ios -maxdepth 1 -name '*.xcworkspace' -print -quit)"
test -n "$workspace" || { echo "iOS workspace missing; run make mobile-prebuild" >&2; exit 1; }
scheme="$(xcodebuild -list -json -workspace "$workspace" | python3 -c 'import json,sys; data=json.load(sys.stdin); schemes=data.get("workspace",{}).get("schemes",[]); sys.stdout.write(schemes[0] if schemes else "")')"
test -n "$scheme" || { echo "iOS application scheme missing" >&2; exit 1; }
xcodebuild -workspace "$workspace" -scheme "$scheme" -configuration Debug -sdk iphonesimulator CODE_SIGNING_ALLOWED=NO build
