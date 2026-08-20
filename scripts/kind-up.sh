#!/usr/bin/env sh
set -eu

cluster_name="observability-lab"
chart_version="88.0.1"

for command in docker kind kubectl helm; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf 'Missing required command: %s\n' "$command" >&2
    exit 1
  fi
done

if ! kind get clusters | grep -qx "$cluster_name"; then
  kind create cluster --config deploy/kubernetes/kind-config.yaml
fi

docker build --build-arg SERVICE=api -t devops-observability-lab/imports-api:local .
docker build --build-arg SERVICE=worker -t devops-observability-lab/imports-worker:local .
docker build --build-arg SERVICE=integration-simulator -t devops-observability-lab/integration-simulator:local .
kind load docker-image --name "$cluster_name" \
  devops-observability-lab/imports-api:local \
  devops-observability-lab/imports-worker:local \
  devops-observability-lab/integration-simulator:local

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts --force-update
helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  --version "$chart_version" \
  --namespace monitoring \
  --create-namespace \
  --values deploy/kubernetes/monitoring/kube-prometheus-values.yaml \
  --wait --timeout 10m

kubectl apply -f deploy/kubernetes/monitoring/telemetry-stack.yaml
kubectl rollout status deployment/loki -n monitoring --timeout=5m
kubectl rollout status deployment/tempo -n monitoring --timeout=5m
kubectl rollout status deployment/otel-collector -n monitoring --timeout=5m

for dashboard in deploy/grafana/dashboards/*.json; do
  name="grafana-$(basename "$dashboard" .json)"
  kubectl create configmap "$name" -n monitoring --from-file="$(basename "$dashboard")=$dashboard" \
    --dry-run=client -o yaml | kubectl apply -f -
  kubectl label configmap "$name" -n monitoring grafana_dashboard=1 --overwrite
done

kubectl apply -k deploy/kubernetes/overlays/local
kubectl apply -f deploy/kubernetes/monitoring/application-monitoring.yaml
kubectl rollout status statefulset/postgres -n observability-lab --timeout=5m
kubectl rollout status statefulset/rabbitmq -n observability-lab --timeout=5m
kubectl rollout status deployment/imports-api -n observability-lab --timeout=5m
kubectl rollout status deployment/imports-worker -n observability-lab --timeout=5m

printf '%s\n' "Kubernetes lab is ready:"
printf '%s\n' "  API:     http://localhost:8080"
printf '%s\n' "  Grafana: http://localhost:3000 (admin / observability_dev)"
