# Two PostgreSQL instances; the second sets log_error_verbosity to terse. .all() must fail.
resource "google_sql_database_instance" "good" {
  name             = "pg-good"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    database_flags {
      name  = "log_error_verbosity"
      value = "verbose"
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
      name  = "log_error_verbosity"
      value = "terse"
    }
  }
}
