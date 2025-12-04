# Create a Cloud SQL SQL Server instance
resource "google_sql_database_instance" "sqlserver_public_instance" {
  name             = "sqlserver-pass-instance"
  region           = var.region
  database_version = "SQLSERVER_2019_EXPRESS" # var.database_version

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
      enabled    = true
      start_time = "00:00"
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

# Create a database within the SQL Server instance
resource "google_sql_database" "sqlserver_database" {
  name     = var.database_name
  instance = google_sql_database_instance.sqlserver_public_instance.name
}

# Create a user for the database
resource "google_sql_user" "sqlserver_user" {
  name     = var.user_name
  instance = google_sql_database_instance.sqlserver_public_instance.name
  password = var.user_password
}