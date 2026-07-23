# Non-compliant: MySQL source profile stores the password inline instead of Secret Manager.
resource "google_datastream_connection_profile" "mysql" {
  display_name          = "mysql-source"
  location              = "us-central1"
  connection_profile_id = "mysql-source"

  mysql_profile {
    hostname = "10.5.0.4"
    port     = 3306
    username = "datastream"
    password = "SuperSecret123!"
  }

  private_connectivity {
    private_connection = "projects/my-project/locations/us-central1/privateConnections/pc-1"
  }
}
