package k8s.externalsecret

is_external_secret {
  input.apiVersion == "external-secrets.io/v1beta1"
  input.kind == "ExternalSecret"
}

deny[msg] {
  is_external_secret
  not input.spec.secretStoreRef
  msg := "ExternalSecret must set spec.secretStoreRef"
}

deny[msg] {
  is_external_secret
  not object.get(input.spec.secretStoreRef, "name", "")
  msg := "ExternalSecret must set spec.secretStoreRef.name"
}

deny[msg] {
  is_external_secret
  not object.get(input.spec.secretStoreRef, "kind", "")
  msg := "ExternalSecret must set spec.secretStoreRef.kind"
}

deny[msg] {
  is_external_secret
  not input.spec.target
  msg := "ExternalSecret must set spec.target"
}

deny[msg] {
  is_external_secret
  object.get(input.spec.target, "name", "") != "app-secrets"
  msg := "ExternalSecret target name must be app-secrets"
}

deny[msg] {
  is_external_secret
  count(object.get(input.spec, "data", [])) == 0
  msg := "ExternalSecret must define at least one spec.data mapping"
}
