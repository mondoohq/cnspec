# Compliant: the rule admits only instances carrying the bastion tag, not the internet.
resource "google_compute_firewall" "rule" {
  name    = "source-tags"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["0-65535"]
  }

  source_tags = ["bastion"]
}
