# Non-compliant: PostgreSQL instance exposes a public IPv4 address.
resource "google_sql_database_instance" "fail_example" {
  name             = "pg-app"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled = true
    }
  }
}
