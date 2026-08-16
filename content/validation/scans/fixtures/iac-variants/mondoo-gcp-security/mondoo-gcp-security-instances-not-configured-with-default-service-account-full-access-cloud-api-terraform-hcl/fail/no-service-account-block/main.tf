# Non-compliant: no service_account block, so the instance falls back to the
# default compute service account with default (broad) scopes.
resource "google_compute_instance" "fail_example" {
  name         = "legacy-server"
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
