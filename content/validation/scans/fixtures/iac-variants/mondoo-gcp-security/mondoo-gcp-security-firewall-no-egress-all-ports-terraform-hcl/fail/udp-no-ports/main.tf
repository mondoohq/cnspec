# Non-compliant: egress to the internet allows UDP on every port (no ports set).
resource "google_compute_firewall" "egress_udp_all" {
  name      = "allow-egress-udp"
  network   = "default"
  direction = "EGRESS"

  allow {
    protocol = "udp"
  }

  destination_ranges = ["::/0"]
}
