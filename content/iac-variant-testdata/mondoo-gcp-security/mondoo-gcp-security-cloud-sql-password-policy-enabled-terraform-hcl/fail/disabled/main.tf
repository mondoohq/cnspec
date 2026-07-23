# Non-compliant: password validation policy block exists but is disabled.
resource "google_sql_database_instance" "fail_example" {
  name             = "app-db"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    password_validation_policy {
      enable_password_policy = false
      min_length             = 8
    }
  }
}
