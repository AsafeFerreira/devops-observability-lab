#!/usr/bin/env sh
set -eu

cluster_name="observability-lab"
if kind get clusters 2>/dev/null | grep -qx "$cluster_name"; then
  kind delete cluster --name "$cluster_name"
  printf 'Deleted kind cluster %s. Its local cluster data is not recoverable.\n' "$cluster_name"
else
  printf 'Cluster %s does not exist; nothing changed.\n' "$cluster_name"
fi
