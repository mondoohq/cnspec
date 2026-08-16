# Compliant: log_connections=on is set via a dynamic "database_flags" block,
# a very common pattern when flags come from a map variable.
variable "db_flags" {
  type = map(string)
  default = {
    log_connections            = "on"
    log_min_duration_statement = "1000"
  }
}

resource "google_sql_database_instance" "pass_dynamic" {
  name             = "pg-app"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    dynamic "database_flags" {
      for_each = var.db_flags
      content {
        name  = database_flags.key
        value = database_flags.value
      }
    }
  }
}
