#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

bash scripts/grafana/generate_dashboards.sh

if ! git diff --quiet -- configs/grafana/dashboards k8s/overlays/observability-base/configmaps/dashboards; then
  echo "ci: generated Grafana dashboards are out of date" >&2
  echo "Run: bash scripts/grafana/generate_dashboards.sh" >&2
  git diff -- configs/grafana/dashboards k8s/overlays/observability-base/configmaps/dashboards >&2
  exit 1
fi

echo "ci: grafana dashboard generation check passed"
