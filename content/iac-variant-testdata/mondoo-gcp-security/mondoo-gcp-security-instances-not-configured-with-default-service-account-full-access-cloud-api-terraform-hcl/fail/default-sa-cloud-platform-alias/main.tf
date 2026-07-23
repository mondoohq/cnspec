# Non-compliant: default compute service account with the cloud-platform scope
# alias grants full API access.
resource "google_compute_instance" "fail_example" {
  name         = "app-server"
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
