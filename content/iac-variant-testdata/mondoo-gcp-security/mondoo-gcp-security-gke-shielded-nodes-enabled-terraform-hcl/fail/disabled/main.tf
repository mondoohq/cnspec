# Non-compliant: Shielded GKE Nodes disabled and no node-level shielding.
resource "google_container_cluster" "primary" {
  name     = "unshielded-cluster"
  location = "us-central1"

  initial_node_count    = 1
  enable_shielded_nodes = false
}
