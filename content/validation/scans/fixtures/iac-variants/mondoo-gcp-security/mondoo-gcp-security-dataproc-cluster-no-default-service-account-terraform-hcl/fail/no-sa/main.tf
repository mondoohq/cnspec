# Non-compliant: no service_account set, so the cluster uses the default Compute Engine SA.
resource "google_dataproc_cluster" "default_sa" {
  name   = "default-sa-cluster"
  region = "us-central1"

  cluster_config {
    gce_cluster_config {
      zone       = "us-central1-a"
      subnetwork = "projects/my-project/regions/us-central1/subnetworks/private-subnet"
    }
  }
}
