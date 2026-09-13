# Non-compliant: UDP carries ports, so an allow block with none opens every UDP port.
resource "google_compute_firewall" "udp_all_ports" {
  name    = "allow-udp-any-port"
  network = "default"

  direction     = "INGRESS"
  source_ranges = ["0.0.0.0/0"]

  allow {
    protocol = "udp"
  }
}
