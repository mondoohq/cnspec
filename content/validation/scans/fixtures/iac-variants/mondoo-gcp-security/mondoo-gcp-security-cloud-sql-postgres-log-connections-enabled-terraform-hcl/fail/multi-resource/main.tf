# Two PostgreSQL instances; the second omits log_connections. .all() must fail.
resource "google_sql_database_instance" "good" {
  name             = "pg-good"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "log_connections"
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
      name  = "log_connections"
      value = "off"
    }
  }
}
