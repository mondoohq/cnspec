# Compliant: log_error_verbosity is set to default.
resource "google_sql_database_instance" "pass_example" {
  name             = "pg-app"
  database_version = "POSTGRES_14"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "log_error_verbosity"
      value = "default"
    }
  }
}
