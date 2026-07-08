# Non-compliant: all TCP ports (incl. 22) open to the internet, no ports set.
resource "google_compute_firewall" "allow_all_tcp" {
  name      = "allow-all-tcp"
  network   = "default"
  direction = "INGRESS"

  allow {
    protocol = "tcp"
  }

  source_ranges = ["0.0.0.0/0"]
}
