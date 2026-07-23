# Compliant: cloud-platform scope is granted, but to a dedicated custom service
# account (not the default compute SA), which is an accepted pattern.
resource "google_compute_instance" "pass_example" {
  name         = "ci-runner"
  machine_type = "e2-standard-2"
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
    email  = "ci-runner-sa@my-project.iam.gserviceaccount.com"
    scopes = ["cloud-platform"]
  }
}
