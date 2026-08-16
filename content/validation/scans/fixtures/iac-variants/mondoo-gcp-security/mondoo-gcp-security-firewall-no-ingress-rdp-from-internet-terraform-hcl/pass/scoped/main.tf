# Compliant: RDP ingress is restricted to a bastion subnet.
resource "google_compute_firewall" "allow_rdp" {
  name    = "allow-rdp"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["3389"]
  }

  source_ranges = ["10.10.0.0/24"]
}
