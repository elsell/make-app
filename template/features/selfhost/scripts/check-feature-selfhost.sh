#!/usr/bin/env bash
set -euo pipefail
python3 scripts/test-selfhost-package.py
python3 scripts/test-selfhost-preflight.py
