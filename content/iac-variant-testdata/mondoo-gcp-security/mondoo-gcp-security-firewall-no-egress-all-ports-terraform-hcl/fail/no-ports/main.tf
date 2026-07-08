# Non-compliant: egress to the internet allows TCP on every port (no ports set).
resource "google_compute_firewall" "egress_tcp_all" {
  name      = "allow-egress-tcp"
  network   = "default"
  direction = "EGRESS"

  allow {
    protocol = "tcp"
  }

  destination_ranges = ["0.0.0.0/0"]
}
