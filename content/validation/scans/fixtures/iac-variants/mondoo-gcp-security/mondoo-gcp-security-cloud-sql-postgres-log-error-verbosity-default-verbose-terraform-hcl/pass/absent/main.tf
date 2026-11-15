# Compliant: no log_error_verbosity flag, so the PostgreSQL default of `default` applies.
resource "google_sql_database_instance" "pass_example" {
  name             = "pg-app"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "log_connections"
      value = "on"
    }
  }
}
