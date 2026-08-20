#!/usr/bin/env sh
set -eu

check() {
  name="$1"
  url="$2"
  if curl --silent --show-error --fail --max-time 5 \
    --retry 20 --retry-delay 1 --retry-all-errors "$url" >/dev/null; then
    printf 'OK   %s\n' "$name"
  else
    printf 'FAIL %s (%s)\n' "$name" "$url" >&2
    return 1
  fi
}

check "Imports API" "http://localhost:8080/health/ready"
check "Imports worker" "http://localhost:18081/health/ready"
check "Integration simulator" "http://localhost:18082/health/ready"
check "Prometheus" "http://localhost:9090/-/ready"
check "Alertmanager" "http://localhost:9093/-/ready"
check "Loki" "http://localhost:3100/ready"
check "Tempo" "http://localhost:3200/ready"
check "OpenTelemetry Collector" "http://localhost:13133/"
check "Grafana" "http://localhost:3000/api/health"
check "RabbitMQ management" "http://localhost:15672/"

docker compose exec -T postgres pg_isready \
  -U "${POSTGRES_USER:-observability}" \
  -d "${POSTGRES_DB:-observability_lab}" >/dev/null
printf 'OK   PostgreSQL\n'
