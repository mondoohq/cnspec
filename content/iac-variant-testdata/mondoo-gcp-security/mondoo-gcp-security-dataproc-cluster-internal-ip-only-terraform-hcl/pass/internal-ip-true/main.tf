# Compliant: gce_cluster_config enables internal_ip_only.
resource "google_dataproc_cluster" "compliant" {
  name   = "secure-cluster"
  region = "us-central1"

  cluster_config {
    gce_cluster_config {
      zone             = "us-central1-a"
      internal_ip_only = true
      subnetwork       = "projects/my-project/regions/us-central1/subnetworks/private-subnet"
    }
  }
}
