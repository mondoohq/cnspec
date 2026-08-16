# Non-compliant: connection profile tunnels through a forward SSH bastion.
resource "google_datastream_connection_profile" "ssh_tunnel" {
  display_name          = "mysql-source"
  location              = "us-central1"
  connection_profile_id = "mysql-source"

  mysql_profile {
    hostname                       = "10.5.0.4"
    port                           = 3306
    username                       = "datastream"
    secret_manager_stored_password = "projects/my-project/secrets/mysql-password/versions/latest"
  }

  forward_ssh_connectivity {
    hostname    = "bastion.example.com"
    username    = "tunnel"
    port        = 22
    private_key = "REDACTED"
  }
}
