# Compliant: custom-mode VPC, auto_create_subnetworks set.
resource "google_compute_network" "vpc" {
  name                    = "prod-vpc"
  auto_create_subnetworks = false
}
