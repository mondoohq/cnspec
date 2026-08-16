resource "google_compute_firewall" "https_v6" {
  name    = "allow-https-v6"
  network = "default"

  direction     = "INGRESS"
  source_ranges = ["::/0"]

  allow {
    protocol = "tcp"
    ports    = ["443"]
  }
}
