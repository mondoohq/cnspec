# Compliant: MySQL source profile references the password from Secret Manager.
resource "google_datastream_connection_profile" "mysql" {
  display_name          = "mysql-source"
  location              = "us-central1"
  connection_profile_id = "mysql-source"

  mysql_profile {
    hostname                       = "10.5.0.4"
    port                           = 3306
    username                       = "datastream"
    secret_manager_stored_password = "projects/my-project/secrets/mysql-password/versions/latest"
  }

  private_connectivity {
    private_connection = "projects/my-project/locations/us-central1/privateConnections/pc-1"
  }
}
