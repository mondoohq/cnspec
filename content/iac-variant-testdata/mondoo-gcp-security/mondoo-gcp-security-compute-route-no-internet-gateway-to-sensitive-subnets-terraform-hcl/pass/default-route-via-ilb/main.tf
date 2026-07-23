# Compliant: default route sends egress to an internal load balancer, not the
# internet gateway.
resource "google_compute_route" "egress" {
  name         = "route-through-ilb"
  network      = google_compute_network.vpc.id
  dest_range   = "0.0.0.0/0"
  next_hop_ilb = google_compute_forwarding_rule.ilb.id
  priority     = 1000
}
