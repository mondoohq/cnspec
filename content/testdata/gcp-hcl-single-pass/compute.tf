# Create a VPC network for the compute instance
resource "google_compute_network" "vpc_network" {
  name                    = "my-vpc-network-${random_id.rnd.hex}"
  auto_create_subnetworks = true
}

# Private Service Access for Cloud SQL private IP connectivity
resource "google_compute_global_address" "private_ip_address" {
  name          = "private-ip-address-${random_id.rnd.hex}"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.vpc_network.id
  depends_on = [ google_compute_network.vpc_network.id ]
}

resource "google_service_networking_connection" "private_vpc_connection" {
  network                 = google_compute_network.vpc_network.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_address.name]
}

resource "google_compute_instance" "default" {
  name         = "my-instance-${random_id.rnd.hex}"
  machine_type = "n2d-standard-2" # n2d required for Confidential VM
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