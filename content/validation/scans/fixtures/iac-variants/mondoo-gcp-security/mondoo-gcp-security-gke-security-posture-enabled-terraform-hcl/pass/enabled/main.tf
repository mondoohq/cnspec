# Compliant: the security posture mode is set to BASIC.
resource "google_container_cluster" "primary" {
  name     = "posture-cluster"
  location = "us-central1"

  initial_node_count = 1

  security_posture_config {
    mode = "BASIC"
  }
}
