resource "google_compute_firewall" "tcp_all_ports" {
  name    = "allow-tcp-any-port"
  network = "default"

  direction     = "INGRESS"
  source_ranges = ["0.0.0.0/0"]

  allow {
    protocol = "tcp"
  }
}
