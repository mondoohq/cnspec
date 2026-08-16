# Compliant: log_lock_waits flag is set to on.
resource "google_sql_database_instance" "pass_example" {
  name             = "pg-app"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "log_lock_waits"
      value = "on"
    }
  }
}
