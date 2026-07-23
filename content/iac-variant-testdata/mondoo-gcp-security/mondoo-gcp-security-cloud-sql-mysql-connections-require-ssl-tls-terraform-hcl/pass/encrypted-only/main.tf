# Compliant: MySQL instance requires TLS via ssl_mode = ENCRYPTED_ONLY.
resource "google_sql_database_instance" "pass_example" {
  name             = "mysql-db"
  database_version = "MYSQL_8_0"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled = false
      ssl_mode     = "ENCRYPTED_ONLY"
    }
  }
}
