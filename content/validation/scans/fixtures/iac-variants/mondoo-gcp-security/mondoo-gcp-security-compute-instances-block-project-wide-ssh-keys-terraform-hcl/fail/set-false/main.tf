# Non-compliant: project-wide SSH keys are explicitly allowed.
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
    block-project-ssh-keys = "false"
  }
}
