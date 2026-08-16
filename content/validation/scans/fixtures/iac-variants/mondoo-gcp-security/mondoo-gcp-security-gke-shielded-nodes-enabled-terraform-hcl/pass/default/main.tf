# Compliant: enable_shielded_nodes not set, defaults to enabled (not false).
resource "google_container_cluster" "primary" {
  name     = "shielded-default-cluster"
  location = "us-central1"

  initial_node_count = 1
}
