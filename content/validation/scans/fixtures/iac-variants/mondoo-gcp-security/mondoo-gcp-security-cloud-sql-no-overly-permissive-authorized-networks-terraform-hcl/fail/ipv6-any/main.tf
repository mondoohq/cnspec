# Non-compliant: one good network plus an all-IPv6 range slips through.
resource "google_sql_database_instance" "fail_example" {
  name             = "app-db"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled = true

      authorized_networks {
        name  = "office"
        value = "203.0.113.0/24"
      }

      authorized_networks {
        name  = "everywhere-v6"
        value = "::/0"
      }
    }
  }
}
