# Compliant: confidential_instance_type selects SEV without the older boolean flag.
resource "google_compute_instance" "example" {
  name         = "confidential-instance"
  machine_type = "n2d-standard-2"
  zone         = "us-central1-a"

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }

  network_interface {
    network = "default"
  }

  confidential_instance_config {
    confidential_instance_type = "SEV"
  }

  scheduling {
    on_host_maintenance = "TERMINATE"
  }
}
