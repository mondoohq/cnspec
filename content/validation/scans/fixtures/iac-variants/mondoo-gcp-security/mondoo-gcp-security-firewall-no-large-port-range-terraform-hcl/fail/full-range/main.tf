# Non-compliant: public ingress opens the entire port range.
resource "google_compute_firewall" "allow_all_ports" {
  name    = "allow-all-ports"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["1-65535"]
  }

  source_ranges = ["0.0.0.0/0"]
}
