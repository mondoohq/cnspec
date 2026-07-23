# Non-compliant: no workload_identity_config block.
resource "google_container_cluster" "primary" {
  name     = "no-wi-cluster"
  location = "us-central1"

  initial_node_count = 1
}
