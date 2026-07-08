# Compliant: all required audit logging flags are set.
resource "google_alloydb_instance" "pass_example" {
  cluster       = google_alloydb_cluster.default.name
  instance_id   = "pass-instance"
  instance_type = "PRIMARY"

  database_flags = {
    "log_connections"           = "on"
    "log_disconnections"        = "on"
    "log_min_duration_statement" = "500"
  }
}
