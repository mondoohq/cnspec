# Non-compliant: the security_posture_config block is present but never sets
# mode, so the cluster leaves the security posture at MODE_UNSPECIFIED. Only
# the retired workload vulnerability scanning setting is configured here.
resource "google_container_cluster" "primary" {
  name     = "posture-unset-cluster"
  location = "us-central1"

  initial_node_count = 1

  security_posture_config {
    vulnerability_mode = "VULNERABILITY_BASIC"
  }
}
