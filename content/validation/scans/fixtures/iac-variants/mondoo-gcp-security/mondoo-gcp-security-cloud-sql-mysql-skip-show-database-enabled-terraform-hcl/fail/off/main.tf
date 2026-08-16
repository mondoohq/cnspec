# Non-compliant: MySQL skip_show_database flag is explicitly set to off.
resource "google_sql_database_instance" "fail_example" {
  name             = "mysql-db"
  database_version = "MYSQL_8_0"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "skip_show_database"
      value = "off"
    }
  }
}
