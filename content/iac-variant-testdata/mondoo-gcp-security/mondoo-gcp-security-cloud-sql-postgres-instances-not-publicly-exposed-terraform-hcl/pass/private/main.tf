# Compliant: PostgreSQL instance has no public IPv4 (private connectivity only).
resource "google_sql_database_instance" "pass_example" {
  name             = "pg-app"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled    = false
      private_network = "projects/example/global/networks/default"
    }
  }
}
