# Non-compliant: the schedule inherits encryption from a database that uses Google-managed keys.
resource "google_spanner_database" "plain" {
  instance = "my-instance"
  name     = "my-database"
}

resource "google_spanner_backup_schedule" "fail_example" {
  instance = "my-instance"
  database = google_spanner_database.plain.name
  name     = "my-schedule"

  retention_duration = "172800s"

  spec {
    cron_spec {
      text = "0 12 * * *"
    }
  }

  full_backup_spec {}

  encryption_config {
    encryption_type = "USE_DATABASE_ENCRYPTION"
  }
}
