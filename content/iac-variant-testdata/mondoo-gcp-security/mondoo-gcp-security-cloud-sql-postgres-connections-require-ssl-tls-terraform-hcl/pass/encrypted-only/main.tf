# Compliant: PostgreSQL instance requires TLS via ssl_mode = ENCRYPTED_ONLY.
resource "google_sql_database_instance" "pass_example" {
  name             = "pg-app"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled = false
      ssl_mode     = "ENCRYPTED_ONLY"
    }
  }
}
