# Non-compliant: no encryption_config block, so backups fall back to defaults.
resource "google_spanner_backup_schedule" "fail_example" {
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
}
