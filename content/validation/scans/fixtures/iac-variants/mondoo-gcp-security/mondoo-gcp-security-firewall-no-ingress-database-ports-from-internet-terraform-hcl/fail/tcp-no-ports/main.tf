# Non-compliant: an allow block with no ports opens every TCP port, database ports included.
resource "google_compute_firewall" "allow_tcp_any_port" {
  name      = "allow-tcp-any-port"
  network   = "default"
  direction = "INGRESS"

  allow {
    protocol = "tcp"
  }

  source_ranges = ["::/0"]
}
