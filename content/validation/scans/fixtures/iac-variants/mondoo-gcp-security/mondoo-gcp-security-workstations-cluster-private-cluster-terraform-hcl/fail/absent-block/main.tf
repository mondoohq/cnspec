# Non-compliant: no private_cluster_config block; cluster is publicly reachable.
resource "google_workstations_workstation_cluster" "example" {
  workstation_cluster_id = "public-cluster"
  network                = "projects/my-project/global/networks/my-vpc"
  subnetwork             = "projects/my-project/regions/us-central1/subnetworks/my-subnet"
  location               = "us-central1"
}
