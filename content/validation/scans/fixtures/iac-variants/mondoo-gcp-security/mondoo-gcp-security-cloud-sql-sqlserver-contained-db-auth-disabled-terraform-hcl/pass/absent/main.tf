# Compliant: no contained database authentication flag, so the Cloud SQL default of off applies.
resource "google_sql_database_instance" "pass_example" {
  name             = "mssql-app"
  database_version = "SQLSERVER_2019_STANDARD"
  region           = "us-central1"
  root_password    = "example-not-a-real-secret"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "cross db ownership chaining"
      value = "off"
    }
  }
}
