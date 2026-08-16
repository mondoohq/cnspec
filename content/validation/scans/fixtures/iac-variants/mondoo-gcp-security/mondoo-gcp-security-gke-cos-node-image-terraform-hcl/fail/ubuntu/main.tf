resource "google_container_cluster" "ubuntu_nodes" {
  name               = "ubuntu-nodes"
  location           = "us-central1"
  initial_node_count = 1

  node_config {
    machine_type = "e2-medium"
    image_type   = "UBUNTU_CONTAINERD"
  }
}
