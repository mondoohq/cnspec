# Compliant: MySQL skip_show_database flag is set to on.
resource "google_sql_database_instance" "pass_example" {
  name             = "mysql-db"
  database_version = "MYSQL_8_0"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "skip_show_database"
      value = "on"
    }
  }
}
