# Compliant: IP forwarding explicitly disabled.
resource "google_compute_instance" "example" {
  name           = "app-instance"
  machine_type   = "e2-medium"
  zone           = "us-central1-a"
  can_ip_forward = false

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }

  network_interface {
    network = "default"
  }
}
