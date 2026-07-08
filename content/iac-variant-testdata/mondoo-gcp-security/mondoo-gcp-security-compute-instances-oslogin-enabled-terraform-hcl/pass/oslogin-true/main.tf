# Compliant: OS Login enabled via metadata.
resource "google_compute_instance" "vm" {
  name         = "app-vm"
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

  metadata = {
    enable-oslogin = "true"
  }
}
