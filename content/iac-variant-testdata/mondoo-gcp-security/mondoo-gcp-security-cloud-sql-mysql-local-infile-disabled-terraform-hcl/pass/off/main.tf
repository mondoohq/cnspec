# Compliant: MySQL local_infile flag is set to off.
resource "google_sql_database_instance" "pass_example" {
  name             = "mysql-db"
  database_version = "MYSQL_8_0"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "local_infile"
      value = "off"
    }
  }
}
