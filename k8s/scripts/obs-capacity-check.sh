#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${K8S_NAMESPACE:-secure-observable}"
MAX_RESTARTS="${OBS_MAX_RESTARTS:-3}"

declare -A MIN_GI=(
  [tempo-data]=2
  [loki-data]=4
  [mimir-data]=5
  [grafana-data]=2
)

to_gi() {
  local raw="$1"
  case "$raw" in
    *Gi) echo "${raw%Gi}" ;;
    *Mi) awk -v mi="${raw%Mi}" 'BEGIN { printf "%.4f", mi/1024 }' ;;
    *Ti) awk -v ti="${raw%Ti}" 'BEGIN { printf "%.4f", ti*1024 }' ;;
    *) echo "0" ;;
  esac
}

if ! kubectl get ns "${NAMESPACE}" >/dev/null 2>&1; then
  echo "obs-capacity-check: namespace '${NAMESPACE}' not found; skipping"
  exit 0
fi

status=0

echo "obs-capacity-check: validating PVC requested capacities in namespace ${NAMESPACE}"
for pvc in "${!MIN_GI[@]}"; do
  requested="$(kubectl -n "${NAMESPACE}" get pvc "${pvc}" -o jsonpath='{.spec.resources.requests.storage}' 2>/dev/null || true)"
  if [[ -z "${requested}" ]]; then
    echo "ERROR: missing PVC '${pvc}'"
    status=1
    continue
  fi

  requested_gi="$(to_gi "${requested}")"
  min_gi="${MIN_GI[$pvc]}"

  if ! awk -v got="${requested_gi}" -v min="${min_gi}" 'BEGIN { exit !(got >= min) }'; then
    echo "ERROR: PVC ${pvc} requested ${requested} (< ${min_gi}Gi minimum)"
    status=1
  else
    echo "OK: PVC ${pvc} requested ${requested} (>= ${min_gi}Gi)"
  fi
done

echo "obs-capacity-check: validating restart/backpressure proxy thresholds"
for app in otel-collector tempo loki mimir grafana; do
  pod_lines="$(kubectl -n "${NAMESPACE}" get pods -l app.kubernetes.io/name="${app}" -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.status.phase}{" "}{range .status.containerStatuses[*]}{.restartCount}{" "}{end}{"\n"}{end}' 2>/dev/null || true)"
  if [[ -z "${pod_lines}" ]]; then
    echo "WARN: no pods found for app '${app}'"
    status=1
    continue
  fi

  while IFS= read -r line; do
    [[ -z "${line}" ]] && continue
    pod_name="$(echo "$line" | awk '{print $1}')"
    phase="$(echo "$line" | awk '{print $2}')"

    if [[ "${phase}" != "Running" ]]; then
      echo "ERROR: pod ${pod_name} (${app}) phase=${phase}"
      status=1
    fi

    restarts="$(echo "$line" | awk '{max=0; for(i=3;i<=NF;i++) if($i>max) max=$i; print max}')"
    if ! awk -v r="${restarts}" -v m="${MAX_RESTARTS}" 'BEGIN { exit !(r <= m) }'; then
      echo "ERROR: pod ${pod_name} (${app}) restartCount=${restarts} (> ${MAX_RESTARTS})"
      status=1
    else
      echo "OK: pod ${pod_name} (${app}) restartCount=${restarts}"
    fi
  done <<< "${pod_lines}"
done

if [[ "${status}" -ne 0 ]]; then
  echo "obs-capacity-check: FAILED"
  exit 1
fi

echo "obs-capacity-check: PASSED"
