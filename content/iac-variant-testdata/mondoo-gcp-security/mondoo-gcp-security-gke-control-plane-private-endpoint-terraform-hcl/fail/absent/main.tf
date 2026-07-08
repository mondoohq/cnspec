resource "google_container_cluster" "no_private_config" {
  name               = "no-private-config"
  location           = "us-central1"
  initial_node_count = 1
}
