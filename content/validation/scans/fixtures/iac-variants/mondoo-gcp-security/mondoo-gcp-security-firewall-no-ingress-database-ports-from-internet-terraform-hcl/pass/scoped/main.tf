# Compliant: MySQL ingress is restricted to the application subnet.
resource "google_compute_firewall" "allow_mysql" {
  name    = "allow-mysql"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["3306"]
  }

  source_ranges = ["10.0.1.0/24"]
}
