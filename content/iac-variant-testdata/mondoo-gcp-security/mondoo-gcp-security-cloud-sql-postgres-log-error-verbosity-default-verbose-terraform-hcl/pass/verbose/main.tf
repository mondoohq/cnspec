# Compliant: log_error_verbosity is set to verbose.
resource "google_sql_database_instance" "pass_example" {
  name             = "pg-app"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "log_error_verbosity"
      value = "verbose"
    }
  }
}
