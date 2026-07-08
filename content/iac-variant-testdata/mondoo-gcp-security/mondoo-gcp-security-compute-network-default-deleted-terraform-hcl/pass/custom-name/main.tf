# Compliant: custom-named VPC, not the auto-created "default" network.
resource "google_compute_network" "vpc" {
  name                    = "prod-vpc"
  auto_create_subnetworks = false
}
