resource "google_container_cluster" "no_netpol" {
  name               = "no-netpol-cluster"
  location           = "us-central1"
  initial_node_count = 1
}
