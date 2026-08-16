resource "google_compute_firewall" "ssh" {
  name    = "allow-ssh"
  network = "default"

  source_ranges = ["0.0.0.0/0"]

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }
}
