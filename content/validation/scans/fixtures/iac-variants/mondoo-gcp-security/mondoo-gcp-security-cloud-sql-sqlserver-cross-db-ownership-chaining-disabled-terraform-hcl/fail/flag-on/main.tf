# Non-compliant: cross db ownership chaining is turned on.
resource "google_sql_database_instance" "sqlserver" {
  name             = "sqlserver-prod"
  database_version = "SQLSERVER_2019_STANDARD"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "cross db ownership chaining"
      value = "on"
    }
  }
}
