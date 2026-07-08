# Two PostgreSQL instances; the second turns log_lock_waits off. .all() must fail.
resource "google_sql_database_instance" "good" {
  name             = "pg-good"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "log_lock_waits"
      value = "on"
    }
  }
}

resource "google_sql_database_instance" "bad" {
  name             = "pg-bad"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "log_lock_waits"
      value = "off"
    }
  }
}
