resource "google_compute_firewall" "v6_all" {
  name    = "allow-v6-all"
  network = "default"

  direction     = "INGRESS"
  source_ranges = ["::/0"]

  allow {
    protocol = "all"
  }
}
