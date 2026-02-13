#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

SCRIPT="scripts/ci/run_k8s_rollout_runtime_kind.sh"

if [[ ! -f "${SCRIPT}" ]]; then
  echo "k8s rollout runtime: missing script ${SCRIPT}" >&2
  exit 1
fi

bash -n "${SCRIPT}"

if ! grep -q "runtime-kind: PASSED" "${SCRIPT}"; then
  echo "k8s rollout runtime: expected success marker not found" >&2
  exit 1
fi

echo "k8s rollout runtime: script sanity passed"
