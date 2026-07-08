# Non-compliant: backup_configuration is present but backups are disabled.
resource "google_sql_database_instance" "fail_example" {
  name             = "app-db"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    backup_configuration {
      enabled = false
    }
  }
}
