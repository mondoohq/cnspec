# Compliant: 10.0.0.0/8 is private address space, not the internet.
resource "google_compute_firewall" "allow_ssh" {
  name    = "allow-ssh-internal"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["10.0.0.0/8"]
}
