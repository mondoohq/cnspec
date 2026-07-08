# Non-compliant: cluster-level flag disabled and node_config only enables Secure
# Boot without integrity monitoring, so the node-level branch is not satisfied.
resource "google_container_cluster" "primary" {
  name     = "partial-shielded-cluster"
  location = "us-central1"

  initial_node_count    = 1
  enable_shielded_nodes = false

  node_config {
    machine_type = "e2-medium"

    shielded_instance_config {
      enable_secure_boot          = true
      enable_integrity_monitoring = false
    }
  }
}
