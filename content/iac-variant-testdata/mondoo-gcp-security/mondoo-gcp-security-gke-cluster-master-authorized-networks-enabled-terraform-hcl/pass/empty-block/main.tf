resource "google_container_cluster" "primary" {
  name               = "primary"
  location           = "us-central1"
  initial_node_count = 1

  master_authorized_networks_config {
    gcp_public_cidrs_access_enabled = false
  }
}
