# Compliant: auto-mode VPC (still not a legacy network) with the field set.
resource "google_compute_network" "vpc" {
  name                    = "auto-vpc"
  auto_create_subnetworks = true
}
