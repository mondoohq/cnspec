# Non-compliant: no private_cluster_config block at all.
resource "google_container_cluster" "primary" {
  name     = "default-cluster"
  location = "us-central1"

  initial_node_count = 1
}
