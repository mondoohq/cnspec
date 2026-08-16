# Non-compliant: log_lock_waits flag is explicitly turned off.
resource "google_sql_database_instance" "fail_example" {
  name             = "pg-app"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "log_lock_waits"
      value = "off"
    }
  }
}
