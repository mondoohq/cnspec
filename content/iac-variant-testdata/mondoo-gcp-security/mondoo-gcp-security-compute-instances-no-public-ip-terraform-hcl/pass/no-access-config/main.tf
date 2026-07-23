# Compliant: network_interface present, no access_config so no public IP.
resource "google_compute_instance" "private" {
  name         = "private-vm"
  machine_type = "e2-medium"
  zone         = "us-central1-a"

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.internal.id
  }
}
