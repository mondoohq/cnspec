# Compliant: backup schedule inherits the database's encryption configuration.
resource "google_spanner_backup_schedule" "pass_example" {
  instance = "my-instance"
  database = "my-database"
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
