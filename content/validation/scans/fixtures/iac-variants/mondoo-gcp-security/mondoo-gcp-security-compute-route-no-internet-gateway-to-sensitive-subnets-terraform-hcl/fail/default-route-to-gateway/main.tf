# Non-compliant: default route to the internet gateway.
resource "google_compute_route" "internet" {
  name             = "route-to-internet"
  network          = google_compute_network.vpc.id
  dest_range       = "0.0.0.0/0"
  next_hop_gateway = "default-internet-gateway"
  priority         = 1000
}
