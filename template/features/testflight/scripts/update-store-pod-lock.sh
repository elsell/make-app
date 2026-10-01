#!/usr/bin/env bash
set -euo pipefail
# Deliberate native dependency review on macOS; restore authored app configuration.
[[ "$(uname -s)" == Darwin ]] || { echo 'Store pod lock generation requires macOS' >&2; exit 1; }
backup="$(mktemp)"
cp apps/mobile/app.json "$backup"
trap 'cp "$backup" apps/mobile/app.json; rm -f "$backup"' EXIT
node scripts/validate-mobile-release-env.mjs
version="$(node -p "JSON.parse(require('fs').readFileSync('apps/mobile/app.json')).expo.version")"
node scripts/prepare-store-build.mjs "v$version" 1
pnpm --dir apps/mobile mobile:prebuild
(cd apps/mobile && bundle install --jobs 4 --retry 3 && cd ios && bundle exec pod install)
cp apps/mobile/ios/Podfile.lock apps/mobile/Podfile.store.lock
echo 'Review and commit apps/mobile/Podfile.store.lock before tagging a store release.'
