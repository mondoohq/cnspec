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
      ssl_mode = "ENCRYPTED_ONLY"

      # Configure authorized networks to restrict access
      # This limits public access to specific IP addresses
      authorized_networks {
        name  = var.authorized_network_name
        value = var.authorized_network_cidr
      }
    }

    # Enable backup configuration
    backup_configuration {
      enabled                        = true
      point_in_time_recovery_enabled = true
      start_time                     = "00:00"
    }

    # Enable security-focused database flags
    # see https://docs.cloud.google.com/sql/docs/postgres/flags
    database_flags {
      name  = "log_connections"
      value = "on"
    }

    database_flags {
      name  = "log_disconnections"
      value = "on"
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

# Create a database within the Postgres instance
resource "google_sql_database" "postgres_database" {
  name     = var.database_name
  instance = google_sql_database_instance.mysql_public_instance.name
}

# Create a user for the database
resource "google_sql_user" "user_postgres" {
  name     = var.user_name
  instance = google_sql_database_instance.mysql_public_instance.name
  password = var.user_password
}