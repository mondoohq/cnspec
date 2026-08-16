# Non-compliant: an EGRESS rule with no destination_ranges applies to 0.0.0.0/0,
# so allowing every protocol reaches the whole internet.
resource "google_compute_firewall" "egress_no_dest" {
  name    = "egress-all-no-dest"
  network = "default"

  direction = "EGRESS"

  allow {
    protocol = "all"
  }
}
