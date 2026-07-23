# Two SQL Server instances; the second exposes a public IPv4 address. .all() must fail.
resource "google_sql_database_instance" "good" {
  name             = "mssql-good"
  database_version = "SQLSERVER_2019_STANDARD"
  region           = "us-central1"
  root_password    = "Str0ng-Passw0rd!"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled = false
    }
  }
}

resource "google_sql_database_instance" "bad" {
  name             = "mssql-bad"
  database_version = "SQLSERVER_2019_STANDARD"
  region           = "us-central1"
  root_password    = "Str0ng-Passw0rd!"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled = true
    }
  }
}
