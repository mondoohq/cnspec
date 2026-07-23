# Non-compliant: PostgreSQL source profile stores the password inline instead of Secret Manager.
resource "google_datastream_connection_profile" "postgres" {
  display_name          = "postgres-source"
  location              = "us-central1"
  connection_profile_id = "postgres-source"

  postgresql_profile {
    hostname = "10.5.0.3"
    port     = 5432
    username = "datastream"
    database = "app"
    password = "SuperSecret123!"
  }

  private_connectivity {
    private_connection = "projects/my-project/locations/us-central1/privateConnections/pc-1"
  }
}
