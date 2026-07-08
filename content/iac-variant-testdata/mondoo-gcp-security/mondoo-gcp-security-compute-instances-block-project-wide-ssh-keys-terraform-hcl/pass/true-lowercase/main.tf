# Compliant: instance blocks project-wide SSH keys via metadata.
resource "google_compute_instance" "example" {
  name         = "app-instance"
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
    block-project-ssh-keys = "true"
  }
}
