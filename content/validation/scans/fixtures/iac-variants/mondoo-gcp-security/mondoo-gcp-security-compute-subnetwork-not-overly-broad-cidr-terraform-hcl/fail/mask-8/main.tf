# Non-compliant: subnetwork uses an overly broad /8 CIDR range.
resource "google_compute_subnetwork" "fail_example" {
  name          = "wide-subnet"
  ip_cidr_range = "10.0.0.0/8"
  region        = "us-central1"
  network       = "my-network"
}
