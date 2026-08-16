# Non-compliant: backup schedule uses Google-default encryption, not CMEK.
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

  encryption_config {
    encryption_type = "GOOGLE_DEFAULT_ENCRYPTION"
  }
}
