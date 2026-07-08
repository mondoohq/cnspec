# Non-compliant: shielded_instance_config present but integrity monitoring disabled.
resource "google_compute_instance" "example" {
  name         = "shielded-off-instance"
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

  shielded_instance_config {
    enable_integrity_monitoring = false
    enable_secure_boot          = true
    enable_vtpm                 = true
  }
}
