# Non-compliant: no security_posture_config block.
resource "google_container_cluster" "primary" {
  name     = "no-posture-cluster"
  location = "us-central1"

  initial_node_count = 1
}
