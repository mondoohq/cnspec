resource "google_compute_firewall" "wide_open" {
  name    = "allow-all"
  network = "default"

  direction     = "INGRESS"
  source_ranges = ["0.0.0.0/0"]

  allow {
    protocol = "all"
  }
}
