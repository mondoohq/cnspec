# Compliant: connector enforcement is REQUIRED on the instance settings.
resource "google_sql_database_instance" "pass_example" {
  name             = "app-db"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier                  = "db-custom-2-7680"
    connector_enforcement = "REQUIRED"

    ip_configuration {
      ipv4_enabled = false
    }
  }
}
