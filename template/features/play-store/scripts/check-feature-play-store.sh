#!/usr/bin/env bash
set -euo pipefail
node --test scripts/release-notes.test.mjs scripts/play-store-notes.test.mjs scripts/google-play-client.test.mjs
