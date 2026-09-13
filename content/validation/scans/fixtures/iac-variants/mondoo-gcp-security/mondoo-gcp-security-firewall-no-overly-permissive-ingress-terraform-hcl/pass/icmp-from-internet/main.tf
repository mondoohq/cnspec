# Compliant: ICMP carries no ports, so there is nothing for the rule to enumerate.
resource "google_compute_firewall" "allow_icmp" {
  name    = "allow-icmp"
  network = "default"

  direction     = "INGRESS"
  source_ranges = ["0.0.0.0/0"]

  allow {
    protocol = "icmp"
  }
}
