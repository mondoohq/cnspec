# Non-compliant: contained database authentication flag is turned on.
resource "google_sql_database_instance" "fail_example" {
  name             = "mssql-app"
  database_version = "SQLSERVER_2019_STANDARD"
  region           = "us-central1"
  root_password    = "example-not-a-real-secret"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "contained database authentication"
      value = "on"
    }
  }
}
