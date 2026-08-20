#!/usr/bin/env sh
set -eu

timeout_seconds="${STACK_TIMEOUT_SECONDS:-180}"
elapsed=0

while [ "$elapsed" -lt "$timeout_seconds" ]; do
  if curl --silent --fail http://localhost:8080/health/ready >/dev/null 2>&1 && \
     curl --silent --fail http://localhost:18081/health/ready >/dev/null 2>&1 && \
     curl --silent --fail http://localhost:18082/health/ready >/dev/null 2>&1 && \
     curl --silent --fail http://localhost:9090/-/ready >/dev/null 2>&1; then
    printf '%s\n' "Application and monitoring stack are ready."
    exit 0
  fi
  sleep 3
  elapsed=$((elapsed + 3))
done

printf '%s\n' "Stack did not become ready in ${timeout_seconds}s." >&2
docker compose ps >&2
exit 1
