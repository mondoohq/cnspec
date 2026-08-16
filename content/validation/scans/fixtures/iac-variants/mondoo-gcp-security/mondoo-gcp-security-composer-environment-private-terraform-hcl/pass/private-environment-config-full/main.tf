# Compliant: config block with a fully specified private_environment_config block.
resource "google_composer_environment" "example" {
  name   = "prod-environment"
  region = "us-central1"

  config {
    private_environment_config {
      enable_private_endpoint                = true
      master_ipv4_cidr_block                 = "172.16.0.0/28"
      cloud_composer_network_ipv4_cidr_block = "172.16.1.0/24"
    }
  }
}
