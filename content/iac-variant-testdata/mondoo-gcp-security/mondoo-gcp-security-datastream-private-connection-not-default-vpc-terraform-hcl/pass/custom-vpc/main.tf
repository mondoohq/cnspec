# Compliant: private connection peers with a dedicated VPC, not the default network.
resource "google_datastream_private_connection" "compliant" {
  display_name          = "datastream-pc"
  location              = "us-central1"
  private_connection_id = "datastream-pc"

  vpc_peering_config {
    vpc    = "projects/my-project/global/networks/datastream-vpc"
    subnet = "10.10.0.0/29"
  }
}
