# Compliant: the recreated default-allow rule is disabled.
resource "google_compute_firewall" "default_allow_ssh" {
  name     = "default-allow-ssh"
  network  = "default"
  disabled = true

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["0.0.0.0/0"]
}
