# Non-compliant: SCTP is port based, so egress to the internet with no ports
# set opens every SCTP port.
resource "google_compute_firewall" "egress_sctp_all" {
  name      = "allow-egress-sctp"
  network   = "default"
  direction = "EGRESS"

  allow {
    protocol = "sctp"
  }

  destination_ranges = ["0.0.0.0/0"]
}
