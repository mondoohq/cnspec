# Non-compliant: connector enforcement is explicitly NOT_REQUIRED.
resource "google_sql_database_instance" "fail_example" {
  name             = "app-db"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled          = false
      connector_enforcement = "NOT_REQUIRED"
    }
  }
}
