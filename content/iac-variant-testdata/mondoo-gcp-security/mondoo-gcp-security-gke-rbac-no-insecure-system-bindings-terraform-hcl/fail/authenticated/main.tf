# Non-compliant: insecure binding for system:authenticated is enabled.
resource "google_container_cluster" "primary" {
  name     = "insecure-auth-cluster"
  location = "us-central1"

  initial_node_count = 1

  rbac_binding_config {
    enable_insecure_binding_system_authenticated   = true
    enable_insecure_binding_system_unauthenticated = false
  }
}
