# ipv4_enabled driven by a ternary whose active branch enables a public IP.
variable "public" {
  type    = bool
  default = true
}

resource "google_sql_database_instance" "fail_ternary" {
  name             = "pg-app"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled = var.public ? true : false
    }
  }
}
