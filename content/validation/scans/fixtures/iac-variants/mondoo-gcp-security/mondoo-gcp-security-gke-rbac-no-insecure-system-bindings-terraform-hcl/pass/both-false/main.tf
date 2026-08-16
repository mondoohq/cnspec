# Compliant: rbac_binding_config disables both insecure system bindings.
resource "google_container_cluster" "primary" {
  name     = "rbac-cluster"
  location = "us-central1"

  initial_node_count = 1

  rbac_binding_config {
    enable_insecure_binding_system_authenticated   = false
    enable_insecure_binding_system_unauthenticated = false
  }
}
