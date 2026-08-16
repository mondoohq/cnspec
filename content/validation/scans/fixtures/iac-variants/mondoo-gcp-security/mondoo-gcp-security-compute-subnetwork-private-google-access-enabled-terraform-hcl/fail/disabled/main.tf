# Non-compliant: subnetwork explicitly disables Private Google Access.
resource "google_compute_subnetwork" "fail_example" {
  name                     = "private-subnet"
  ip_cidr_range            = "10.0.1.0/24"
  region                   = "us-central1"
  network                  = "my-network"
  private_ip_google_access = false
}
