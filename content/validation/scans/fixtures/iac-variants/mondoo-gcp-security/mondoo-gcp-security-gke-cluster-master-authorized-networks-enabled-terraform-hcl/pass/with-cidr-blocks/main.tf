resource "google_container_cluster" "primary" {
  name               = "primary"
  location           = "us-central1"
  initial_node_count = 1

  master_authorized_networks_config {
    cidr_blocks {
      cidr_block   = "10.0.0.0/8"
      display_name = "corp-network"
    }
  }
}
