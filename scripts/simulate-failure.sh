#!/usr/bin/env sh
set -eu

scenario="${1:-help}"
base_url="${BASE_URL:-http://localhost:8080}"
run_id="$(date +%s)-$$"

submit() {
  source="$1"
  curl --silent --show-error --fail-with-body \
    -X POST "${base_url}/api/v1/imports" \
    -H 'Content-Type: application/json' \
    -H 'X-Tenant-ID: client-a' \
    -H "X-Correlation-ID: failure-${run_id}" \
    -H "Idempotency-Key: failure-${source}-${run_id}" \
    --data "{\"source\":\"${source}\",\"recordCount\":25}"
  printf '\n'
}

case "$scenario" in
  integration) submit force-error ;;
  latency) submit slow ;;
  flaky) submit flaky ;;
  stop-worker) docker compose stop imports-worker ;;
  recover-worker) docker compose start imports-worker ;;
  stop-database) docker compose stop postgres ;;
  recover-database) docker compose start postgres ;;
  evidence) docker compose exec -T alert-recorder sh -c 'test -f /data/alerts.ndjson && tail -n 20 /data/alerts.ndjson || true' ;;
  help|*)
    printf '%s\n' "Usage: $0 {integration|latency|flaky|stop-worker|recover-worker|stop-database|recover-database|evidence}"
    ;;
esac
