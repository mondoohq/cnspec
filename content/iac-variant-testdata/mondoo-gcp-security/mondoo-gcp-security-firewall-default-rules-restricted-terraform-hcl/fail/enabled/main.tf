# Non-compliant: default-allow rule is explicitly enabled.
resource "google_compute_firewall" "default_allow_ssh" {
  name     = "default-allow-ssh"
  network  = "default"
  disabled = false

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["0.0.0.0/0"]
}
