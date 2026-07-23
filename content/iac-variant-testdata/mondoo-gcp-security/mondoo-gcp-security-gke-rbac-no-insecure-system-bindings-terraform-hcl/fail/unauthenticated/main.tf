# Non-compliant: insecure binding for system:unauthenticated is enabled.
resource "google_container_cluster" "primary" {
  name     = "insecure-unauth-cluster"
  location = "us-central1"

  initial_node_count = 1

  rbac_binding_config {
    enable_insecure_binding_system_authenticated   = false
    enable_insecure_binding_system_unauthenticated = true
  }
}
