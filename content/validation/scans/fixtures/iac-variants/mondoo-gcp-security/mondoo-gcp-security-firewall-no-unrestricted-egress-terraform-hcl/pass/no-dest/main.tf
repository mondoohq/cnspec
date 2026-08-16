resource "google_compute_firewall" "egress_no_dest" {
  name    = "egress-all-no-dest"
  network = "default"

  direction = "EGRESS"

  allow {
    protocol = "all"
  }
}
