# Compliant: security posture and vulnerability scanning are both enabled.
resource "google_container_cluster" "primary" {
  name     = "posture-cluster"
  location = "us-central1"

  initial_node_count = 1

  security_posture_config {
    mode               = "BASIC"
    vulnerability_mode = "VULNERABILITY_BASIC"
  }
}
