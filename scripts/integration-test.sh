#!/usr/bin/env sh
set -eu

base_url="${BASE_URL:-http://localhost:8080}"
tenant="${TEST_TENANT:-client-a}"
run_id="$(date +%s)-$$"

json_field() {
  field="$1"
  python3 -c 'import json,sys; print(json.load(sys.stdin)[sys.argv[1]])' "$field"
}

create_import() {
  source="$1"
  idempotency_key="$2"
  curl --silent --show-error --fail-with-body \
    -X POST "${base_url}/api/v1/imports" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: ${tenant}" \
    -H "X-Correlation-ID: integration-${run_id}" \
    -H "Idempotency-Key: ${idempotency_key}" \
    --data "{\"source\":\"${source}\",\"recordCount\":25}"
}

wait_for_status() {
  import_id="$1"
  expected="$2"
  attempts=0
  while [ "$attempts" -lt 40 ]; do
    body="$(curl --silent --show-error --fail-with-body \
      -H "X-Tenant-ID: ${tenant}" \
      "${base_url}/api/v1/imports/${import_id}")"
    status="$(printf '%s' "$body" | json_field status)"
    if [ "$status" = "$expected" ]; then
      printf 'OK   import %s reached %s\n' "$import_id" "$expected"
      return 0
    fi
    sleep 1
    attempts=$((attempts + 1))
  done
  printf 'FAIL import %s did not reach %s\n' "$import_id" "$expected" >&2
  return 1
}

success_key="success-${run_id}"
success_body="$(create_import normal "$success_key")"
success_id="$(printf '%s' "$success_body" | json_field id)"
wait_for_status "$success_id" SUCCEEDED

replay_body="$(create_import normal "$success_key")"
replay_id="$(printf '%s' "$replay_body" | json_field id)"
if [ "$success_id" != "$replay_id" ]; then
  printf '%s\n' "FAIL idempotency returned a different import ID" >&2
  exit 1
fi
printf 'OK   idempotency preserved import %s\n' "$success_id"

failure_body="$(create_import force-error "failure-${run_id}")"
failure_id="$(printf '%s' "$failure_body" | json_field id)"
wait_for_status "$failure_id" DEAD_LETTER

printf '%s\n' "Integration scenarios passed."
