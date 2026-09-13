# Compliant: the range 3307-5431 sits between MySQL and PostgreSQL and covers no database port.
resource "google_compute_firewall" "allow_app_range" {
  name      = "allow-app-range"
  network   = "default"
  direction = "INGRESS"

  allow {
    protocol = "tcp"
    ports    = ["3307-5431"]
  }

  source_ranges = ["0.0.0.0/0"]
}
