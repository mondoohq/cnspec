# Compliant: ESP carries no ports, so there is nothing for the rule to enumerate.
resource "google_compute_firewall" "egress_esp" {
  name      = "allow-egress-esp"
  network   = "default"
  direction = "EGRESS"

  allow {
    protocol = "esp"
  }

  destination_ranges = ["0.0.0.0/0"]
}
