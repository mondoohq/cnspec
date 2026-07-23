# Compliant: SQL Server instance has no public IPv4 (private connectivity only).
resource "google_sql_database_instance" "pass_example" {
  name             = "mssql-app"
  database_version = "SQLSERVER_2019_STANDARD"
  region           = "us-central1"
  root_password    = "example-not-a-real-secret"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled    = false
      private_network = "projects/example/global/networks/default"
    }
  }
}
