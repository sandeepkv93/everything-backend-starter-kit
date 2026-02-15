#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${K8S_NAMESPACE:-everything-backend}"
GRAFANA_PORT="${GRAFANA_LOCAL_PORT:-13000}"
GRAFANA_URL="http://127.0.0.1:${GRAFANA_PORT}"
GRAFANA_USER="${GRAFANA_USER:-admin}"
GRAFANA_PASSWORD="${GRAFANA_PASSWORD:-admin}"
EVIDENCE_DIR="${EVIDENCE_DIR:-.artifacts/k8s-rollout-evidence}"

mkdir -p "${EVIDENCE_DIR}"

for tool in kubectl curl jq; do
  if ! command -v "${tool}" >/dev/null 2>&1; then
    echo "grafana-runtime-check: missing required tool: ${tool}" >&2
    exit 1
  fi
done

kubectl -n "${NAMESPACE}" port-forward svc/grafana "${GRAFANA_PORT}:3000" >"${EVIDENCE_DIR}/portforward-grafana.log" 2>&1 &
pf_pid=$!
cleanup() {
  kill "${pf_pid}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

for _ in $(seq 1 40); do
  if curl -fsS "${GRAFANA_URL}/api/health" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

curl -fsS -u "${GRAFANA_USER}:${GRAFANA_PASSWORD}" "${GRAFANA_URL}/api/health" >"${EVIDENCE_DIR}/grafana-health.json"
curl -fsS -u "${GRAFANA_USER}:${GRAFANA_PASSWORD}" "${GRAFANA_URL}/api/search?query=" >"${EVIDENCE_DIR}/grafana-search.json"

for uid in api-overview logs-overview trace-overview; do
  if ! jq -e --arg uid "${uid}" '.[] | select(.uid == $uid)' "${EVIDENCE_DIR}/grafana-search.json" >/dev/null; then
    echo "grafana-runtime-check: missing dashboard uid ${uid}" >&2
    exit 1
  fi
done

for ds in mimir loki tempo; do
  curl -fsS -u "${GRAFANA_USER}:${GRAFANA_PASSWORD}" "${GRAFANA_URL}/api/datasources/uid/${ds}" >"${EVIDENCE_DIR}/grafana-datasource-${ds}.json"
done

echo "grafana-runtime-check: PASSED"
