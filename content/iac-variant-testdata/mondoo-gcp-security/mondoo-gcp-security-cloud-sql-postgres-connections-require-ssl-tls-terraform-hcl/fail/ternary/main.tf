# ssl_mode chosen by a ternary whose active branch permits unencrypted traffic.
variable "strict_tls" {
  type    = bool
  default = false
}

resource "google_sql_database_instance" "fail_ternary" {
  name             = "pg-app"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled = false
      ssl_mode     = var.strict_tls ? "ENCRYPTED_ONLY" : "ALLOW_UNENCRYPTED_AND_ENCRYPTED"
    }
  }
}
