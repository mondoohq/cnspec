resource "google_container_cluster" "open" {
  name               = "open-cluster"
  location           = "us-central1"
  initial_node_count = 1
}
