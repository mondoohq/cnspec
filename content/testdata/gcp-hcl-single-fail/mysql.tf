# Create a Cloud SQL MySQL instance
resource "google_sql_database_instance" "mysql_public_instance" {
  name             = "mysql-fail-instance-${random_id.suffix.hex}"
  region           = var.region
  database_version = "MYSQL_8_0" # var.database_version

  settings {
    tier = var.tier



    # Configure IP connectivity - public IP enabled
    # Note: mondoo-gcp-security-cloud-sql-mysql-instances-not-publicly-exposed-terraform-hcl fails if this block is commented out.
    #       This is because the default behavior is to have a public IP.
    # Based on the documentation, if the ip_configuration subblock exists but ipv4_enabled is not explicitly set, the behavior is:
    # ipv4_enabled defaults to true when not specified.
    ip_configuration {
      ipv4_enabled = true # Enable public IP

      // SSL connection encryption
      ssl_mode = "ALLOW_UNENCRYPTED_AND_ENCRYPTED"

      # Configure authorized networks to restrict access
      # This limits public access to specific IP addresses
      authorized_networks {
        name  = var.authorized_network_name
        value = var.authorized_network_cidr
      }
    }

    # Enable backup configuration
    backup_configuration {
      enabled            = true
      binary_log_enabled = false
      start_time         = "00:00"
    }

    # Enable maintenance window
    maintenance_window {
      day          = 7 # Sunday
      hour         = 2 # 2 AM
      update_track = "stable"
    }
  }

  # Prevent accidental deletion
  deletion_protection = var.deletion_protection
}

# Create a database within the MySQL instance
resource "google_sql_database" "mysql_database" {
  name     = var.database_name
  instance = google_sql_database_instance.mysql_public_instance.name
}

# Create a user for the database
resource "google_sql_user" "mysql_user" {
  name     = var.user_name
  instance = google_sql_database_instance.mysql_public_instance.name
  password = var.user_password
}