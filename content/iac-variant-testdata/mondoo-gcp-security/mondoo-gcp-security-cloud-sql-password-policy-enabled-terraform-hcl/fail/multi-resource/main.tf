# Two instances; the second lacks a password validation policy. .all() must fail.
resource "google_sql_database_instance" "good" {
  name             = "app-db-good"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    password_validation_policy {
      enable_password_policy = true
      min_length             = 12
    }
  }
}

resource "google_sql_database_instance" "bad" {
  name             = "app-db-bad"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"
  }
}
