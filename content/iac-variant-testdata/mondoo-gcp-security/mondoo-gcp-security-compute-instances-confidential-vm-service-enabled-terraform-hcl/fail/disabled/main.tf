# Non-compliant: confidential_instance_config present but disabled.
resource "google_compute_instance" "example" {
  name         = "confidential-off-instance"
  machine_type = "n2d-standard-2"
  zone         = "us-central1-a"

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }

  network_interface {
    network = "default"
  }

  confidential_instance_config {
    enable_confidential_compute = false
  }
}
