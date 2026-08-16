# Compliant: public ingress opens a small, bounded port range.
resource "google_compute_firewall" "allow_app" {
  name    = "allow-app-range"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["8000-9000"]
  }

  source_ranges = ["0.0.0.0/0"]
}
