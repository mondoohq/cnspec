# Compliant: the rule admits only instances carrying the bastion tag, not the internet.
resource "google_compute_firewall" "rule" {
  name    = "source-tags"
  network = "default"

  allow {
    protocol = "all"
  }

  source_tags = ["bastion"]
}
