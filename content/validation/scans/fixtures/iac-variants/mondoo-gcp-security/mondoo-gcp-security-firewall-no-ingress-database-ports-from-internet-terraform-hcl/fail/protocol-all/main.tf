# Non-compliant: protocol all from the internet reaches every database port.
resource "google_compute_firewall" "allow_everything" {
  name      = "allow-everything"
  network   = "default"
  direction = "INGRESS"

  allow {
    protocol = "all"
  }

  source_ranges = ["0.0.0.0/0"]
}
