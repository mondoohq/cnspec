# Non-compliant: 0.0.0.0/1 plus 128.0.0.0/1 is the whole IPv4 internet written as two ranges.
resource "google_compute_firewall" "halves" {
  name    = "halves"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["5432"]
  }

  source_ranges = ["0.0.0.0/1", "128.0.0.0/1"]
}
