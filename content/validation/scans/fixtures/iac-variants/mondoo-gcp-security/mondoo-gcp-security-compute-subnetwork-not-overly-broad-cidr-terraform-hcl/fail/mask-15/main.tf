# Non-compliant: subnetwork uses an overly broad /15 CIDR range.
resource "google_compute_subnetwork" "fail_example" {
  name          = "wide-subnet"
  ip_cidr_range = "10.0.0.0/15"
  region        = "us-central1"
  network       = "my-network"
}
