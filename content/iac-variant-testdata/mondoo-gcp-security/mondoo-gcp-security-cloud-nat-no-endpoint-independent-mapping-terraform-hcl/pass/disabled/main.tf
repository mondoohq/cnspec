# Compliant: endpoint-independent mapping is explicitly disabled.
resource "google_compute_router_nat" "pass_example" {
  name                               = "app-nat"
  router                             = "app-router"
  region                             = "us-central1"
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  enable_endpoint_independent_mapping = false
}
