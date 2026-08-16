# Compliant: subnetwork uses a /16 CIDR range (mask >= 16).
resource "google_compute_subnetwork" "pass_example" {
  name          = "app-subnet"
  ip_cidr_range = "172.16.0.0/16"
  region        = "us-central1"
  network       = "my-network"
}
