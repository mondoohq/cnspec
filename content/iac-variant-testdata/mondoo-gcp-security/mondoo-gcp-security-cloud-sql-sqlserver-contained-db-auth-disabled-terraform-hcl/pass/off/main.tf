# Compliant: contained database authentication flag is set to off.
resource "google_sql_database_instance" "pass_example" {
  name             = "mssql-app"
  database_version = "SQLSERVER_2019_STANDARD"
  region           = "us-central1"
  root_password    = "example-not-a-real-secret"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "contained database authentication"
      value = "off"
    }
  }
}
