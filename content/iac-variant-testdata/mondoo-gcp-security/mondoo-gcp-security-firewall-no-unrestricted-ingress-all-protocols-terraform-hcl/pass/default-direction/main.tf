resource "google_compute_firewall" "ingress_all_rfc1918" {
  name    = "ingress-all-rfc1918"
  network = "default"

  source_ranges = ["172.16.0.0/12"]

  allow {
    protocol = "all"
  }
}
