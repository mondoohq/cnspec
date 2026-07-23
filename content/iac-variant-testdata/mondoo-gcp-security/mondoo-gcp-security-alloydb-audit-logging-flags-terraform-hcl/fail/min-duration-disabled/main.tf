# Non-compliant: log_min_duration_statement is -1 (disabled).
resource "google_alloydb_instance" "fail_example" {
  cluster       = google_alloydb_cluster.default.name
  instance_id   = "fail-instance"
  instance_type = "PRIMARY"

  database_flags = {
    "log_connections"           = "on"
    "log_disconnections"        = "on"
    "log_min_duration_statement" = "-1"
  }
}
