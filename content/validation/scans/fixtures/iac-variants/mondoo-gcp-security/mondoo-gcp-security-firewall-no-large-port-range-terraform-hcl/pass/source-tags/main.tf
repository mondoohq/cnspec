# Compliant: the check limits port ranges only on rules open to the internet, and this
# rule admits only instances carrying the bastion tag.
resource "google_compute_firewall" "rule" {
  name    = "source-tags"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["0-65535"]
  }

  source_tags = ["bastion"]
}
