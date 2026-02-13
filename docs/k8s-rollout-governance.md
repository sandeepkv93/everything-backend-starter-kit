# Kubernetes Rollout Governance

This document defines ownership and promotion criteria for the optional Argo Rollouts blue/green path.

## Scope
- Overlay: `k8s/overlays/rollouts/blue-green`
- Commands:
  - `task k8s:deploy-rollout-bluegreen`
  - `task k8s:rollout-status`
  - `task k8s:rollout-promote`
  - `task k8s:rollout-abort`
  - `task k8s:rollout-promote-production ALLOW_PROD_ROLLOUTS=true`
  - `task k8s:rollout-abort-production ALLOW_PROD_ROLLOUTS=true`

## Ownership Model
- Platform/SRE ownership:
  - Argo Rollouts controller and CRD lifecycle (install/upgrade/backup/rollback).
  - Cluster-level RBAC and plugin availability guidance.
  - CI validation gate maintenance for rollout policy (`run_k8s_rollout_validate.sh`).
- Application team ownership:
  - Rollout manifest strategy parameters (replicas, promotion behavior, health probes).
  - Service-level readiness/liveness correctness.
  - Execution of staging and production promote/abort flows.

## Environment Policy
- Staging is the default proving ground for rollout operations.
- Production promote/abort requires explicit confirmation via `ALLOW_PROD_ROLLOUTS=true`.
- Production operations must occur only in approved change windows.

## SLO-Linked Promotion Criteria
Before promotion from preview to active, all must pass:
1. Rollout health:
- `task k8s:rollout-status` reports healthy rollout (not degraded).
2. Service health:
- `/health/live` and `/health/ready` pass for preview pods.
3. Stability gate:
- No CrashLoopBackOff pods in rollout workload.
- Max restart count threshold within policy (`<= 1` recommended for production window).
4. Error budget guard:
- No active P1/P2 incident related to API availability.
- Observability panels show no sustained error-rate regression during canary/preview bake window.

## Recommended Bake Windows
- Staging: minimum 10 minutes under representative traffic.
- Production: minimum 20 minutes under representative traffic and no SLO breach indicators.

## Rollback Conditions
Abort rollout immediately when one or more holds:
- readiness/liveness failures after rollout update,
- restart spikes beyond threshold,
- elevated 5xx/error ratio relative to baseline,
- latency regression that threatens SLO.

## Audit Trail Requirements
Record for each production promote/abort:
- operator identity,
- rollout revision hash,
- promotion timestamp,
- precheck evidence references (health + observability snapshots),
- outcome and any follow-up actions.
