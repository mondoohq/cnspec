resource "google_compute_firewall" "ingress_all_anywhere" {
  name    = "ingress-all-anywhere"
  network = "default"

  direction     = "INGRESS"
  source_ranges = ["0.0.0.0/0"]

  allow {
    protocol = "all"
  }
}
