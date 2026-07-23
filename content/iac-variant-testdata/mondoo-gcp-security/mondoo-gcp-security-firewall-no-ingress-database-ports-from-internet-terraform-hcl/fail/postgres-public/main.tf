# Non-compliant: PostgreSQL (port 5432) is open to the entire internet.
resource "google_compute_firewall" "allow_postgres" {
  name      = "allow-postgres"
  network   = "default"
  direction = "INGRESS"

  allow {
    protocol = "tcp"
    ports    = ["5432"]
  }

  source_ranges = ["0.0.0.0/0"]
}
