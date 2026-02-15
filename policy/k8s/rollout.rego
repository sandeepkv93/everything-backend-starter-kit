package k8s.rollout

is_rollout {
  input.apiVersion == "argoproj.io/v1alpha1"
  input.kind == "Rollout"
}

deny[msg] {
  is_rollout
  object.get(input.metadata, "name", "") != "everything-backend-api"
  msg := "Rollout name must be everything-backend-api"
}

deny[msg] {
  is_rollout
  object.get(input.spec, "replicas", 0) < 2
  msg := "Rollout replicas must be >= 2"
}

deny[msg] {
  is_rollout
  not input.spec.strategy.blueGreen
  msg := "Rollout must configure blueGreen strategy"
}

deny[msg] {
  is_rollout
  bg := input.spec.strategy.blueGreen
  object.get(bg, "activeService", "") != "everything-backend-api"
  msg := "Rollout blueGreen.activeService must be everything-backend-api"
}

deny[msg] {
  is_rollout
  bg := input.spec.strategy.blueGreen
  object.get(bg, "previewService", "") != "everything-backend-api-preview"
  msg := "Rollout blueGreen.previewService must be everything-backend-api-preview"
}

deny[msg] {
  is_rollout
  bg := input.spec.strategy.blueGreen
  object.get(bg, "autoPromotionEnabled", true) != false
  msg := "Rollout must set blueGreen.autoPromotionEnabled=false for manual promotion control"
}
