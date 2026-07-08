# Compliant: connection profile uses private connectivity, no forward SSH tunnel.
resource "google_datastream_connection_profile" "compliant" {
  display_name          = "postgres-source"
  location              = "us-central1"
  connection_profile_id = "postgres-source"

  postgresql_profile {
    hostname                       = "10.5.0.3"
    port                           = 5432
    username                       = "datastream"
    database                       = "app"
    secret_manager_stored_password = "projects/my-project/secrets/pg-password/versions/latest"
  }

  private_connectivity {
    private_connection = "projects/my-project/locations/us-central1/privateConnections/pc-1"
  }
}
