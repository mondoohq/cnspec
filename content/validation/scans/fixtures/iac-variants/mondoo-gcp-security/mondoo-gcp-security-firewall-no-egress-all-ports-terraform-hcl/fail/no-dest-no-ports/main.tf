# Non-compliant: no destination_ranges means 0.0.0.0/0, and the allow block
# names no ports, so every TCP port is reachable on the internet.
resource "google_compute_firewall" "egress_open" {
  name    = "egress-open"
  network = "default"

  direction = "EGRESS"

  allow {
    protocol = "tcp"
  }
}
