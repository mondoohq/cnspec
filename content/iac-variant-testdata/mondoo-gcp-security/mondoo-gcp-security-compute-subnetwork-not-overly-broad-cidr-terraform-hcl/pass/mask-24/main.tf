# Compliant: subnetwork uses a narrow /24 CIDR range.
resource "google_compute_subnetwork" "pass_example" {
  name          = "app-subnet"
  ip_cidr_range = "10.0.1.0/24"
  region        = "us-central1"
  network       = "my-network"
}
