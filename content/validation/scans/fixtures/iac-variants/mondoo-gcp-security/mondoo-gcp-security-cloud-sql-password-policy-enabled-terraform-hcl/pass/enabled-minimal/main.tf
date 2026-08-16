# Compliant: minimal password validation policy, only the enable flag set.
resource "google_sql_database_instance" "pass_example" {
  name             = "mysql-app"
  database_version = "MYSQL_8_0"
  region           = "us-central1"

  settings {
    tier = "db-n1-standard-1"

    password_validation_policy {
      enable_password_policy = true
    }
  }
}
