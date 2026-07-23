# Non-compliant: subnetwork has no log_config block, so flow logs are disabled.
resource "google_compute_subnetwork" "fail_example" {
  name          = "private-subnet"
  ip_cidr_range = "10.0.1.0/24"
  region        = "us-central1"
  network       = "my-network"
}
