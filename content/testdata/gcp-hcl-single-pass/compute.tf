# Create a VPC network for the compute instance
resource "google_compute_network" "vpc_network" {
  name                    = "my-vpc-network"
  auto_create_subnetworks = true
}

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
    network = google_compute_network.vpc_network.id
  }

  confidential_instance_config {
    enable_confidential_compute = true
  }

  // see https://docs.cloud.google.com/compute/docs/metadata/predefined-metadata-keys
  metadata = {
    foo = "bar"
    block-project-ssh-keys = "TRUE"
    enable-oslogin = "TRUE"
  }

  metadata_startup_script = "echo hi > /test.txt"
}