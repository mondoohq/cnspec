# Compliant: no network_config block, so no public IP is enabled.
resource "google_alloydb_instance" "pass_example" {
  cluster       = google_alloydb_cluster.default.name
  instance_id   = "pass-instance"
  instance_type = "PRIMARY"
}
