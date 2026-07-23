# A counted instance with the password policy explicitly disabled. Each
# instance violates, so .all() must fail.
resource "google_sql_database_instance" "fail_count" {
  count            = 2
  name             = "app-db-${count.index}"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    password_validation_policy {
      enable_password_policy = false
    }
  }
}
