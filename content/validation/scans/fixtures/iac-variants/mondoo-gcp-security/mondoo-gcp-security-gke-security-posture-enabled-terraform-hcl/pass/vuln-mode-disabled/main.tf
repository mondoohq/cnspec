# Compliant: the security posture mode is set to BASIC. Workload vulnerability
# scanning was retired by Google and is no longer part of this check.
resource "google_container_cluster" "primary" {
  name     = "vuln-disabled-cluster"
  location = "us-central1"

  initial_node_count = 1

  security_posture_config {
    mode               = "BASIC"
    vulnerability_mode = "VULNERABILITY_DISABLED"
  }
}
