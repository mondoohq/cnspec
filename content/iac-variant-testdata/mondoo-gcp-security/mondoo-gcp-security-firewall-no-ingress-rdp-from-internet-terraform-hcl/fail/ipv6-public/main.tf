# Non-compliant: RDP open to the entire IPv6 internet.
resource "google_compute_firewall" "allow_rdp_v6" {
  name      = "allow-rdp-v6"
  network   = "default"
  direction = "INGRESS"

  allow {
    protocol = "tcp"
    ports    = ["3389"]
  }

  source_ranges = ["::/0"]
}
