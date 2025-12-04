resource "google_compute_instance" "default" {
  name         = "my-instance"
  machine_type = "n2-standard-2"
  zone         = "us-central1-a"

  tags = ["foo", "bar"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-11"
      labels = {
        my_label = "value"
      }
    }
  }

  // Local SSD disk
  scratch_disk {
    interface = "NVME"
  }

  network_interface {
    network = "default"

    access_config {
      // Ephemeral public IP
    }
  }

  // see https://docs.cloud.google.com/compute/docs/metadata/predefined-metadata-keys
  metadata = {
    foo = "bar"
    block-project-ssh-keys = "TRUE"
    enable-oslogin = "TRUE"
  }

  metadata_startup_script = "echo hi > /test.txt"
}