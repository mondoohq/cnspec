# Compliant: MySQL instance has no public IPv4 address.
resource "google_sql_database_instance" "pass_example" {
  name             = "mysql-db"
  database_version = "MYSQL_8_0"
  region           = "us-central1"

  settings {
    tier = "db-custom-2-7680"

    ip_configuration {
      ipv4_enabled    = false
      private_network = "projects/my-project/global/networks/my-vpc"
    }
  }
}
