# Non-compliant: instance has no confidential_instance_config block.
resource "google_compute_instance" "example" {
  name         = "standard-instance"
  machine_type = "e2-medium"
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
