resource "google_container_cluster" "primary" {
  name               = "primary"
  location           = "us-central1"
  initial_node_count = 1

  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = true
    master_ipv4_cidr_block  = "172.16.0.0/28"
  }
}
