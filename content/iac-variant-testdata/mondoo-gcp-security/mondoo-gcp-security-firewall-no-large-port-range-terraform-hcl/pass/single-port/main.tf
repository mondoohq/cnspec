# Compliant: public ingress opens a single port, no wide range.
resource "google_compute_firewall" "allow_https" {
  name      = "allow-https"
  network   = "default"
  direction = "INGRESS"

  allow {
    protocol = "tcp"
    ports    = ["443"]
  }

  source_ranges = ["0.0.0.0/0"]
}
