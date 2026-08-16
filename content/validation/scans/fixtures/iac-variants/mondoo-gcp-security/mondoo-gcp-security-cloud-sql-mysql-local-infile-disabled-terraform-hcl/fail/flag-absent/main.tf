# Non-compliant: the local_infile flag is not set, leaving the risky default.
resource "google_sql_database_instance" "fail_example" {
  name             = "mysql-db"
  database_version = "MYSQL_8_0"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "slow_query_log"
      value = "on"
    }
  }
}
