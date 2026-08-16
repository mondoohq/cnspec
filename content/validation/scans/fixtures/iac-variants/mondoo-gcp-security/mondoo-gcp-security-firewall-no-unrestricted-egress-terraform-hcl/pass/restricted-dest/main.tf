resource "google_compute_firewall" "egress_internal" {
  name    = "egress-internal-all"
  network = "default"

  direction          = "EGRESS"
  destination_ranges = ["10.0.0.0/8"]

  allow {
    protocol = "all"
  }
}
