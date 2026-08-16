# Non-compliant: MySQL instance permits unencrypted connections.
resource "google_sql_database_instance" "fail_example" {
  name             = "mysql-db"
  database_version = "MYSQL_8_0"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled = false
      ssl_mode     = "ALLOW_UNENCRYPTED_AND_ENCRYPTED"
    }
  }
}
