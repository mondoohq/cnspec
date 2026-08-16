# Non-compliant: connection profile reaches the source over the public internet via static service IPs.
resource "google_datastream_connection_profile" "static_ip" {
  display_name          = "mysql-source"
  location              = "us-central1"
  connection_profile_id = "mysql-source"

  mysql_profile {
    hostname                       = "203.0.113.10"
    port                           = 3306
    username                       = "datastream"
    secret_manager_stored_password = "projects/my-project/secrets/mysql-password/versions/latest"
  }

  static_service_ip_connectivity {}
}
