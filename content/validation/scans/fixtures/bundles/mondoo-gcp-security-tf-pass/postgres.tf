# Create a Cloud SQL PostgresSQL instance
resource "google_sql_database_instance" "postgres_public_instance" {
  name             = "postgres-pass-instance-${random_id.rnd.hex}"
  region           = var.region
  database_version = "POSTGRES_15" # var.database_version

  depends_on = [google_service_networking_connection.private_vpc_connection]

  settings {
    tier = "db-f1-micro" # PostgreSQL requires shared-core or custom tier


    # Configure IP connectivity - private IP only
    ip_configuration {
      ipv4_enabled    = false
      private_network = google_compute_network.vpc_network.id

      // SSL connection encryption
      ssl_mode = "ENCRYPTED_ONLY"

      // Require Cloud SQL connectors
      connector_enforcement = "REQUIRED"
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

    database_flags {
      name  = "log_lock_waits"
      value = "on"
    }

    # Enable password validation policy
    password_validation_policy {
      enable_password_policy = true
    }

    # Enable maintenance window
    maintenance_window {
      day          = 7 # Sunday
      hour         = 2 # 2 AM
      update_track = "stable"
    }
  }

  encryption_key_name = google_kms_crypto_key.key.id

  # Prevent accidental deletion
  deletion_protection = var.deletion_protection
}

# Create a database within the PostgreSQL instance
resource "google_sql_database" "postgres_database" {
  name     = var.database_name
  instance = google_sql_database_instance.postgres_public_instance.name
}

# Create an IAM user for the database (CLOUD_IAM_USER satisfies the
# cloud-sql-users-use-iam-auth check; password is omitted for IAM users).
resource "google_sql_user" "postgres_user" {
  name     = var.user_name
  instance = google_sql_database_instance.postgres_public_instance.name
  type     = "CLOUD_IAM_USER"
}