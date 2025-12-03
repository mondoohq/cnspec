# Create a Cloud SQL PostgresSQL instance
resource "google_sql_database_instance" "postgres_public_instance" {
  name             = "postgres-pass-instance"
  region           = var.region
  database_version = "POSTGRES_15" # var.database_version

  settings {
    tier = var.tier



    # Configure IP connectivity - public IP enabled
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
      binary_log_enabled = true
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

# Create a database within the PostgreSQL instance
resource "google_sql_database" "postgres_database" {
  name     = var.database_name
  instance = google_sql_database_instance.postgres_public_instance.name
}

# Create a user for the database
resource "google_sql_user" "postgres_user" {
  name     = var.user_name
  instance = google_sql_database_instance.postgres_public_instance.name
  password = var.user_password
}