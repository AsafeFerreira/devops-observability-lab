#!/usr/bin/env sh
set -eu

mkdir -p artifacts
docker compose --profile tools run --rm k6 run /scripts/imports.js
printf '%s\n' "k6 report saved to artifacts/k6-summary.json"
