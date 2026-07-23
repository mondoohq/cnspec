# Non-compliant: security posture mode is DISABLED.
resource "google_container_cluster" "primary" {
  name     = "posture-disabled-cluster"
  location = "us-central1"

  initial_node_count = 1

  security_posture_config {
    mode               = "DISABLED"
    vulnerability_mode = "VULNERABILITY_BASIC"
  }
}
