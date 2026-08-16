# Compliant: connector enforcement is REQUIRED in the ip_configuration block.
resource "google_sql_database_instance" "pass_example" {
  name             = "app-db"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled          = false
      connector_enforcement = "REQUIRED"
    }
  }
}
