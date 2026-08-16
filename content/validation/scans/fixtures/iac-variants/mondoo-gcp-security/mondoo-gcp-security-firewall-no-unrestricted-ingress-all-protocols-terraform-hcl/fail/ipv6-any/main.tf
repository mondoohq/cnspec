resource "google_compute_firewall" "ingress_all_anywhere_v6" {
  name    = "ingress-all-anywhere-v6"
  network = "default"

  direction     = "INGRESS"
  source_ranges = ["::/0"]

  allow {
    protocol = "all"
  }
}
