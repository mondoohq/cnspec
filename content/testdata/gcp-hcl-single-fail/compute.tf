# Create a VPC network for the compute instance
resource "google_compute_network" "vpc_network" {
  name                    = "my-vpc-network-${random_id.suffix.hex}"
  auto_create_subnetworks = true
}

resource "google_compute_instance" "default" {
  name         = "my-instance-${random_id.suffix.hex}"
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

    # mondoo-gcp-security-compute-instances-no-public-ip-terraform-hcl also fails if access_config block is commented out!
    access_config {
      // Ephemeral public IP
    }
  }

  // see https://docs.cloud.google.com/compute/docs/metadata/predefined-metadata-keys
  metadata = {
    foo = "bar"
    block-project-ssh-keys = "FALSE"
    #enable-oslogin = "FALSE"
  }

  metadata_startup_script = "echo hi > /test.txt"
}

resource "google_storage_bucket" "no-public-access" {
  name          = "fail-public-access-bucket-${random_id.suffix.hex}"
  location      = "US"
  force_destroy = true

  #public_access_prevention = "enforced"
}