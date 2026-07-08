# Compliant: no rbac_binding_config block, so no insecure bindings are enabled.
resource "google_container_cluster" "primary" {
  name     = "default-rbac-cluster"
  location = "us-central1"

  initial_node_count = 1
}
