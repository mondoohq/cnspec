# Non-compliant: network_config enables a public IP.
resource "google_alloydb_instance" "fail_example" {
  cluster       = google_alloydb_cluster.default.name
  instance_id   = "fail-instance"
  instance_type = "PRIMARY"

  network_config {
    enable_public_ip = true
  }
}
