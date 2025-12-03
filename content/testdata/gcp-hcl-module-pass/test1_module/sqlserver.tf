# Create a Cloud SQL SQL Server instance
resource "google_sql_database_instance" "sqlserver_public_instance" {
  name             = "sqlserver-pass-instance"
  region           = var.region
  database_version = "SQLSERVER_2019_EXPRESS" # var.database_version

  settings {
    tier = var.tier

    // SSL connection encryption
    ssl_mode = "ENCRYPTED_ONLY"

    # Configure IP connectivity - public IP enabled
    ip_configuration {
      ipv4_enabled = false # Enable public IP

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
      binary_log_enabled = true
      start_time         = "00:00"
    }

    # Enable security-focused database flags
    # see https://docs.cloud.google.com/sql/docs/sqlserver/flags
    database_flags {
      name  = "local_infile"
      value = "off"
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

# Create a database within the SQLServer instance
resource "google_sql_database" "database_sqlserver" {
  name     = var.database_name
  instance = google_sql_database_instance.mysql_public_instance.name
}

# Create a user for the database
resource "google_sql_user" "user_sqlserver" {
  name     = var.user_name
  instance = google_sql_database_instance.mysql_public_instance.name
  password = var.user_password
}