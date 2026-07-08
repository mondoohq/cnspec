# Two PostgreSQL instances; the second exposes a public IPv4 address. .all() must fail.
resource "google_sql_database_instance" "good" {
  name             = "pg-good"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled = false
    }
  }
}

resource "google_sql_database_instance" "bad" {
  name             = "pg-bad"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled = true
    }
  }
}
