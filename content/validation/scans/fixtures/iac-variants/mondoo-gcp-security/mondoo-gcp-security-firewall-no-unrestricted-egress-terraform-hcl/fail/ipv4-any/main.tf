resource "google_compute_firewall" "egress_any" {
  name    = "egress-all-anywhere"
  network = "default"

  direction          = "EGRESS"
  destination_ranges = ["0.0.0.0/0"]

  allow {
    protocol = "all"
  }
}
