#!/usr/bin/env bash
set -euo pipefail
node --test scripts/prepare-store-build.test.mjs scripts/test-testflight-notes.mjs scripts/test-ios-profile-maintenance.mjs scripts/test-apple-maintenance-client.mjs
python3 scripts/test-store-profile.py
