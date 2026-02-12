package k8s.clustersecretstore

is_cluster_secret_store {
  input.apiVersion == "external-secrets.io/v1beta1"
  input.kind == "ClusterSecretStore"
}

deny[msg] {
  is_cluster_secret_store
  object.get(input.metadata, "name", "") != "secure-observable-secret-store"
  msg := "ClusterSecretStore name must be secure-observable-secret-store"
}

deny[msg] {
  is_cluster_secret_store
  not input.spec.provider
  msg := "ClusterSecretStore must define spec.provider"
}

deny[msg] {
  is_cluster_secret_store
  provider := input.spec.provider
  count(provider) == 0
  msg := "ClusterSecretStore provider block must not be empty"
}

deny[msg] {
  is_cluster_secret_store
  provider := input.spec.provider
  count(provider) > 1
  msg := "ClusterSecretStore must define exactly one provider"
}
