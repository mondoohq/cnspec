# Non-compliant: private connection peers with the default VPC network.
resource "google_datastream_private_connection" "default_vpc" {
  display_name          = "datastream-pc"
  location              = "us-central1"
  private_connection_id = "datastream-pc"

  vpc_peering_config {
    vpc    = "projects/my-project/global/networks/default"
    subnet = "10.10.0.0/29"
  }
}
