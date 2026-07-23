resource "google_container_cluster" "primary" {
  name               = "primary"
  location           = "us-central1"
  initial_node_count = 1

  node_config {
    machine_type = "e2-medium"
    image_type   = "COS_CONTAINERD"
  }
}
