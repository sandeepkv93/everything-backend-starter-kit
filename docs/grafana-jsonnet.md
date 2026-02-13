# Grafana Dashboards as Code (Jsonnet)

Grafana dashboard JSON in this repo is generated from Jsonnet sources.

## Source of truth
- `configs/grafana/jsonnet/**`

## Generated outputs
- `configs/grafana/dashboards/*.json`
- `k8s/overlays/observability-base/configmaps/dashboards/*.json`

Do not hand-edit generated JSON files.

## Generate dashboards
```bash
bash scripts/grafana/generate_dashboards.sh
```

## Validate generation in CI style
```bash
bash scripts/ci/run_grafana_dashboards_generate_check.sh
```

## Structure
- `lib/queries/*.libsonnet`: query definitions
- `lib/panels.libsonnet`: panel constructors
- `lib/dashboard.libsonnet`: dashboard defaults
- `dashboards/*.jsonnet`: per-dashboard assembly
- `render.jsonnet`: emits filename->dashboard map

## Pinned dependency
Grafonnet is pinned via:
- `configs/grafana/jsonnet/jsonnetfile.json`
- `configs/grafana/jsonnet/jsonnetfile.lock.json`
