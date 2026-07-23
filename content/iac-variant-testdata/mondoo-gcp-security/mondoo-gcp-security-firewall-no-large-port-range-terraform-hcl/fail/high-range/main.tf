# Non-compliant: public ingress opens a wide high port range up to 65535.
resource "google_compute_firewall" "allow_high_ports" {
  name    = "allow-high-ports"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["1024-65535"]
  }

  source_ranges = ["0.0.0.0/0"]
}
