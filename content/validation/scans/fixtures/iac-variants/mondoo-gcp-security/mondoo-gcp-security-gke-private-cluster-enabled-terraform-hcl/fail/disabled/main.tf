# Non-compliant: private_cluster_config present but private nodes disabled.
resource "google_container_cluster" "primary" {
  name     = "public-nodes-cluster"
  location = "us-central1"

  initial_node_count = 1

  private_cluster_config {
    enable_private_nodes    = false
    enable_private_endpoint = false
  }
}
