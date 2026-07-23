# Compliant: authorized networks are scoped to specific office/VPN CIDRs.
resource "google_sql_database_instance" "pass_example" {
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
        name  = "vpn"
        value = "198.51.100.10/32"
      }
    }
  }
}
