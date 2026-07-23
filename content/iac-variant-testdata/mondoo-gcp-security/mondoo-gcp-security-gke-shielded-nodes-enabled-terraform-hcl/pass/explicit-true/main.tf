# Compliant: Shielded GKE Nodes explicitly enabled at the cluster level.
resource "google_container_cluster" "primary" {
  name     = "shielded-enabled-cluster"
  location = "us-central1"

  initial_node_count   = 1
  enable_shielded_nodes = true
}
