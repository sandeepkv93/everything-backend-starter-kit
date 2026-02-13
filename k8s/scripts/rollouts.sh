#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${K8S_NAMESPACE:-secure-observable}"
ROLLOUT_NAME="${ROLLOUT_NAME:-secure-observable-api}"
ACTION="${1:-status}"

has_rollouts_plugin() {
  kubectl argo rollouts version >/dev/null 2>&1
}

case "${ACTION}" in
  status)
    if has_rollouts_plugin; then
      kubectl argo rollouts get rollout "${ROLLOUT_NAME}" -n "${NAMESPACE}"
    else
      echo "rollouts plugin not found; falling back to kubectl get rollout"
      kubectl -n "${NAMESPACE}" get rollout "${ROLLOUT_NAME}" -o wide
    fi
    ;;
  promote)
    if ! has_rollouts_plugin; then
      echo "kubectl argo rollouts plugin is required for promote action" >&2
      exit 1
    fi
    kubectl argo rollouts promote "${ROLLOUT_NAME}" -n "${NAMESPACE}"
    ;;
  abort)
    if ! has_rollouts_plugin; then
      echo "kubectl argo rollouts plugin is required for abort action" >&2
      exit 1
    fi
    kubectl argo rollouts abort "${ROLLOUT_NAME}" -n "${NAMESPACE}"
    ;;
  *)
    echo "Usage: $0 [status|promote|abort]" >&2
    exit 1
    ;;
esac
