# Non-compliant: no log_connections database flag is defined.
resource "google_sql_database_instance" "fail_example" {
  name             = "pg-app"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "log_min_duration_statement"
      value = "1000"
    }
  }
}
