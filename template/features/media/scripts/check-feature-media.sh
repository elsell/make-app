#!/usr/bin/env bash
set -euo pipefail
(cd modules/media && GOWORK=off go test -race ./...)
