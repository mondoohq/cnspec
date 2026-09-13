# Non-compliant: the range 3000-4000 covers MySQL on 3306 and is open to the internet.
resource "google_compute_firewall" "allow_app_range" {
  name      = "allow-app-range"
  network   = "default"
  direction = "INGRESS"

  allow {
    protocol = "tcp"
    ports    = ["3000-4000"]
  }

  source_ranges = ["0.0.0.0/0"]
}
