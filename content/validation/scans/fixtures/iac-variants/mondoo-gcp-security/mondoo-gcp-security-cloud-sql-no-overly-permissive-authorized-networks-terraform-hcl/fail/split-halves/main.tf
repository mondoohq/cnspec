# Non-compliant: two /1 ranges together cover the entire IPv4 internet.
resource "google_sql_database_instance" "fail_example" {
  name             = "app-db"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled = true

      authorized_networks {
        name  = "lower-half"
        value = "0.0.0.0/1"
      }

      authorized_networks {
        name  = "upper-half"
        value = "128.0.0.0/1"
      }
    }
  }
}
