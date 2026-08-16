# Non-compliant: the skip_show_database flag is not set.
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
