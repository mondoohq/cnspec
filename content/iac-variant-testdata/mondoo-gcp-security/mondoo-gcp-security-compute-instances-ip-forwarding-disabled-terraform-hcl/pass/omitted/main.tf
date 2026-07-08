# Compliant: can_ip_forward omitted, so it defaults to false.
resource "google_compute_instance" "example" {
  name         = "web-instance"
  machine_type = "e2-small"
  zone         = "us-central1-a"

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }

  network_interface {
    network = "default"
  }
}
