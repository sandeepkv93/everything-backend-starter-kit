#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
JSONNET_DIR="${ROOT_DIR}/configs/grafana/jsonnet"
OUT_DIR_1="${ROOT_DIR}/configs/grafana/dashboards"
OUT_DIR_2="${ROOT_DIR}/k8s/overlays/observability-base/configmaps/dashboards"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "grafana-generate: missing required tool: $1" >&2
    exit 1
  fi
}

ensure_deps() {
  if [[ ! -d "${JSONNET_DIR}/vendor" ]]; then
    echo "grafana-generate: vendor tree missing, bootstrapping with jb"
    (
      cd "${JSONNET_DIR}"
      if ! command -v jb >/dev/null 2>&1; then
        echo "grafana-generate: installing jb"
        GOBIN="${ROOT_DIR}/bin" go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@v0.6.0
        export PATH="${ROOT_DIR}/bin:${PATH}"
      fi
      jb install
    )
  fi
}

jsonnet_eval() {
  if command -v go-jsonnet >/dev/null 2>&1; then
    go-jsonnet -J "${JSONNET_DIR}/vendor" "${JSONNET_DIR}/render.jsonnet"
  elif command -v jsonnet >/dev/null 2>&1; then
    jsonnet -J "${JSONNET_DIR}/vendor" "${JSONNET_DIR}/render.jsonnet"
  else
    go run github.com/google/go-jsonnet/cmd/jsonnet@v0.20.0 -J "${JSONNET_DIR}/vendor" "${JSONNET_DIR}/render.jsonnet"
  fi
}

require_cmd jq
require_cmd go
ensure_deps

mkdir -p "${OUT_DIR_1}" "${OUT_DIR_2}"

rendered_json="$(mktemp /tmp/grafana-render.XXXXXX.json)"
jsonnet_eval >"${rendered_json}"

for name in $(jq -r 'keys[]' "${rendered_json}"); do
  jq --arg name "${name}" '.[$name]' "${rendered_json}" >"${OUT_DIR_1}/${name}"
  jq --arg name "${name}" '.[$name]' "${rendered_json}" >"${OUT_DIR_2}/${name}"
done

rm -f "${rendered_json}"

echo "grafana-generate: dashboards generated"
