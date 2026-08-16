# Non-compliant: IP forwarding is enabled.
resource "google_compute_instance" "example" {
  name           = "router-instance"
  machine_type   = "e2-medium"
  zone           = "us-central1-a"
  can_ip_forward = true

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }

  network_interface {
    network = "default"
  }
}
