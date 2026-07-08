resource "google_compute_firewall" "ingress_all_internal" {
  name    = "ingress-all-internal"
  network = "default"

  direction     = "INGRESS"
  source_ranges = ["10.0.0.0/8"]

  allow {
    protocol = "all"
  }
}
