# Non-compliant: private_cluster_config present but private endpoint disabled.
resource "google_workstations_workstation_cluster" "example" {
  workstation_cluster_id = "half-private-cluster"
  network                = "projects/my-project/global/networks/my-vpc"
  subnetwork             = "projects/my-project/regions/us-central1/subnetworks/my-subnet"
  location               = "us-central1"

  private_cluster_config {
    enable_private_endpoint = false
  }
}
