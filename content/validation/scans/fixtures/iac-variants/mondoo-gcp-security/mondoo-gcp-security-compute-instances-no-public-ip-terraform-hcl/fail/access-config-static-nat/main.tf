# Non-compliant: access_config assigns a static external IP.
resource "google_compute_instance" "public" {
  name         = "public-vm"
  machine_type = "e2-medium"
  zone         = "us-central1-a"

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.internal.id

    access_config {
      nat_ip = google_compute_address.static.address
    }
  }
}
