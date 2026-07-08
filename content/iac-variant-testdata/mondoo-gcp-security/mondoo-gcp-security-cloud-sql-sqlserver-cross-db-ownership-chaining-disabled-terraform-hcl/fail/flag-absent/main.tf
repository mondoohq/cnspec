# Non-compliant: the cross db ownership chaining flag is not set at all.
resource "google_sql_database_instance" "sqlserver" {
  name             = "sqlserver-prod"
  database_version = "SQLSERVER_2019_STANDARD"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "contained database authentication"
      value = "off"
    }
  }
}
