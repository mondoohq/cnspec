# Compliant: cluster-level flag disabled, but node_config enables Secure Boot
# and integrity monitoring directly via shielded_instance_config.
resource "google_container_cluster" "primary" {
  name     = "shielded-nodepool-cluster"
  location = "us-central1"

  initial_node_count    = 1
  enable_shielded_nodes = false

  node_config {
    machine_type = "e2-medium"

    shielded_instance_config {
      enable_secure_boot          = true
      enable_integrity_monitoring = true
    }
  }
}
