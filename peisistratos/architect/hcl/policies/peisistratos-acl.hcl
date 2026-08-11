# Reconcile the policies managed by Peisistratos.
path "sys/policies/acl/solon" {
  capabilities = ["create", "read", "update"]
}

path "sys/policies/acl/peisistratos" {
  capabilities = ["create", "read", "update"]
}

# Discover and, during bootstrap, enable the Kubernetes auth mount.
path "sys/auth" {
  capabilities = ["read", "sudo"]
}

path "sys/auth/kubernetes" {
  capabilities = ["create", "read", "update", "sudo"]
}

# Reconcile the Kubernetes auth backend and the two managed roles.
path "auth/kubernetes/config" {
  capabilities = ["create", "read", "update"]
}

path "auth/kubernetes/role/solon" {
  capabilities = ["create", "read", "update"]
}

path "auth/kubernetes/role/peisistratos" {
  capabilities = ["create", "read", "update"]
}
