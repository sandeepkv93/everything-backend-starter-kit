#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

SCRIPT="k8s/scripts/obs-alert-check.sh"

if [[ ! -f "${SCRIPT}" ]]; then
  echo "k8s obs alert check: missing script ${SCRIPT}" >&2
  exit 1
fi

bash -n "${SCRIPT}"

if ! grep -q "obs-alert-check: PASSED" "${SCRIPT}"; then
  echo "k8s obs alert check: expected success marker not found" >&2
  exit 1
fi

echo "k8s obs alert check: script sanity passed"
