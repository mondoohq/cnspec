# Non-compliant: auto_create_subnetworks omitted (legacy network).
resource "google_compute_network" "legacy" {
  name = "legacy-vpc"
}
