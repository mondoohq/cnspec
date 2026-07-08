# Non-compliant: instance uses the default Compute Engine service account.
resource "google_compute_instance" "fail_example" {
  name         = "fail-example"
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
