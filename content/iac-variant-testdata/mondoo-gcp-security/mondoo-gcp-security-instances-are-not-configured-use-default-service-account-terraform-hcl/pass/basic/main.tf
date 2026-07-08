# Compliant: instance uses a dedicated, non-default service account.
resource "google_compute_instance" "pass_example" {
  name         = "pass-example"
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
    email  = "dedicated-sa@my-project.iam.gserviceaccount.com"
    scopes = ["cloud-platform"]
  }
}
