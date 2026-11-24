# Non-compliant: a custom IPv6 default route sends ::/0 to the internet gateway.
resource "google_compute_route" "v6_egress" {
  name             = "v6-egress"
  network          = "default"
  dest_range       = "::/0"
  next_hop_gateway = "default-internet-gateway"
  priority         = 900
}
