# Non-compliant: MySQL instance has no ip_configuration, so ssl_mode is unset.
resource "google_sql_database_instance" "fail_example" {
  name             = "mysql-db"
  database_version = "MYSQL_8_0"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"
  }
}
