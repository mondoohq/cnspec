# Create a Cloud SQL PostgresSQL instance
resource "google_sql_database_instance" "postgres_public_instance" {
  name             = "postgres-pass-instance-${random_id.rnd.hex}"
  region           = var.region
  database_version = "POSTGRES_15" # var.database_version

  depends_on = [google_service_networking_connection.private_vpc_connection]

  settings {
    tier = var.tier


    # Configure IP connectivity - private IP only
    ip_configuration {
      ipv4_enabled    = false
      private_network = google_compute_network.vpc_network.id

      // SSL connection encryption
      ssl_mode = "ENCRYPTED_ONLY"
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

    database_flags {
      name  = "log_error_verbosity"
      value = "default"
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