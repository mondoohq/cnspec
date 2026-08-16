# Compliant: password validation policy is present and enabled.
resource "google_sql_database_instance" "pass_example" {
  name             = "app-db"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    password_validation_policy {
      enable_password_policy      = true
      min_length                  = 12
      complexity                  = "COMPLEXITY_DEFAULT"
      reuse_interval              = 5
      disallow_username_substring = true
    }
  }
}
