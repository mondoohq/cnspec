# Create a Cloud SQL SQL Server instance
resource "google_sql_database_instance" "sqlserver_public_instance" {
  name             = "sqlserver-pass-instance-${random_id.rnd.hex}"
  region           = var.region
  database_version = "SQLSERVER_2019_EXPRESS" # var.database_version
  root_password    = var.user_password

  depends_on = [google_service_networking_connection.private_vpc_connection]

  settings {
    tier = "db-custom-1-3840" # SQL Server requires custom tier


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
      enabled    = true
      start_time = "00:00"
    }

    # Enable password validation policy
    password_validation_policy {
      enable_password_policy = true
    }

    # Disable contained database authentication
    database_flags {
      name  = "contained database authentication"
      value = "off"
    }

    # Disable cross db ownership chaining
    database_flags {
      name  = "cross db ownership chaining"
      value = "off"
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

# Create a database within the SQL Server instance
resource "google_sql_database" "sqlserver_database" {
  name     = var.database_name
  instance = google_sql_database_instance.sqlserver_public_instance.name
}

# SQL Server does not support CLOUD_IAM_* user types; the runtime and
# Terraform variants of cloud-sql-users-use-iam-auth exempt the engine-managed
# "sqlserver" system account. Use that name here so the BUILT_IN user passes.
resource "google_sql_user" "sqlserver_user" {
  name     = "sqlserver"
  instance = google_sql_database_instance.sqlserver_public_instance.name
  password = var.user_password
}