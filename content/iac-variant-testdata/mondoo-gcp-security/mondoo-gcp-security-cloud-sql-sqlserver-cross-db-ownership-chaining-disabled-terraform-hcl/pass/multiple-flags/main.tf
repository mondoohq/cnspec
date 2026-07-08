# Compliant: chaining off alongside other hardening flags.
resource "google_sql_database_instance" "sqlserver" {
  name             = "sqlserver-hardened"
  database_version = "SQLSERVER_2022_ENTERPRISE"
  region           = "us-east1"

  settings {
    tier = "db-custom-4-15360"

    database_flags {
      name  = "contained database authentication"
      value = "off"
    }
    database_flags {
      name  = "cross db ownership chaining"
      value = "off"
    }
  }
}
