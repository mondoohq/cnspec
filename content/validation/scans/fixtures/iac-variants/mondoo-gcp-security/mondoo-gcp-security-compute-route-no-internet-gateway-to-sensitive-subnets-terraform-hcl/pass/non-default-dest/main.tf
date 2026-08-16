# Compliant: internet-gateway route is scoped to a specific range, not 0.0.0.0/0.
resource "google_compute_route" "peer" {
  name             = "route-to-peer"
  network          = google_compute_network.vpc.id
  dest_range       = "10.20.0.0/16"
  next_hop_gateway = "default-internet-gateway"
  priority         = 1000
}
