# Argo Rollouts Blue/Green Overlay (Optional)

This overlay provides an optional Argo Rollouts blue/green path for the API.

## Prerequisites
- Argo Rollouts CRDs/controller installed in cluster.
- `kubectl-argo-rollouts` plugin installed for promotion commands.

## Apply
```bash
kubectl apply -k k8s/overlays/rollouts/blue-green
```

## Operate
```bash
task k8s:rollout-status
task k8s:rollout-promote
task k8s:rollout-abort
```
