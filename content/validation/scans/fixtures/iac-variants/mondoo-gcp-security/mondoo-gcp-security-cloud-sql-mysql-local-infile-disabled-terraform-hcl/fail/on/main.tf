# Non-compliant: MySQL local_infile flag is explicitly set to on.
resource "google_sql_database_instance" "fail_example" {
  name             = "mysql-db"
  database_version = "MYSQL_8_0"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "local_infile"
      value = "on"
    }
  }
}
