# Compliant: egress to the internet is limited to specific TCP ports.
resource "google_compute_firewall" "egress_https" {
  name      = "allow-egress-https"
  network   = "default"
  direction = "EGRESS"

  allow {
    protocol = "tcp"
    ports    = ["443"]
  }

  destination_ranges = ["0.0.0.0/0"]
}
