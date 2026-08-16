resource "google_compute_firewall" "egress_any_v6" {
  name    = "egress-all-anywhere-v6"
  network = "default"

  direction          = "EGRESS"
  destination_ranges = ["10.0.0.0/8", "::/0"]

  allow {
    protocol = "all"
  }
}
