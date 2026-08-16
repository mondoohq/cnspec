# Compliant: network_config present with public IP explicitly disabled.
resource "google_alloydb_instance" "pass_example" {
  cluster       = google_alloydb_cluster.default.name
  instance_id   = "pass-instance"
  instance_type = "PRIMARY"

  network_config {
    enable_public_ip = false
  }
}
