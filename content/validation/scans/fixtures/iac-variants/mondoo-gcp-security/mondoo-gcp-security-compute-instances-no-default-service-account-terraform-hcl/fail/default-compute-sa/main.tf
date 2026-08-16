# Non-compliant: instance explicitly attaches the default compute service account.
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

  service_account {
    email  = "123456789012-compute@developer.gserviceaccount.com"
    scopes = ["cloud-platform"]
  }
}
