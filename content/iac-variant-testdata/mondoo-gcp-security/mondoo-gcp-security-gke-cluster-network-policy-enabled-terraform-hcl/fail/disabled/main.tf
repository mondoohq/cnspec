resource "google_container_cluster" "primary" {
  name               = "primary"
  location           = "us-central1"
  initial_node_count = 1

  network_policy {
    enabled = false
  }
}
