# Non-compliant: egress to the internet allows all protocols and ports.
resource "google_compute_firewall" "egress_all" {
  name      = "allow-egress-all"
  network   = "default"
  direction = "EGRESS"

  allow {
    protocol = "all"
  }

  destination_ranges = ["0.0.0.0/0"]
}
