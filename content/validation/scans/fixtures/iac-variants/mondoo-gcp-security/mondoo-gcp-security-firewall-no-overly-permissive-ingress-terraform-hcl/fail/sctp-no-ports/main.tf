# Non-compliant: SCTP is port based, so an allow block with no ports opens every SCTP port.
resource "google_compute_firewall" "sctp_all_ports" {
  name    = "allow-sctp-any-port"
  network = "default"

  direction     = "INGRESS"
  source_ranges = ["0.0.0.0/0"]

  allow {
    protocol = "sctp"
  }
}
