#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${K8S_NAMESPACE:-secure-observable}"

kubectl get ns "${NAMESPACE}" >/dev/null 2>&1 || {
  echo "Namespace '${NAMESPACE}' not found"
  exit 0
}

kubectl -n "${NAMESPACE}" get deploy,svc,pods | grep -E "NAME|otel-collector|tempo|loki|mimir|grafana" || true
