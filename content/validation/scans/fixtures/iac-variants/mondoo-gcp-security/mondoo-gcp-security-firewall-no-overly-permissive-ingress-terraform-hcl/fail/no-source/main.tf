# Non-compliant: an ingress rule with no source_ranges, source_tags, or source_service_accounts admits 0.0.0.0/0.
resource "google_compute_firewall" "rule" {
  name    = "no-source"
  network = "default"

  allow {
    protocol = "tcp"
  }
}
