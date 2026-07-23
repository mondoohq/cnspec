# Compliant: gce_cluster_config uses a dedicated, non-default service account.
resource "google_dataproc_cluster" "compliant" {
  name   = "secure-cluster"
  region = "us-central1"

  cluster_config {
    gce_cluster_config {
      zone            = "us-central1-a"
      service_account = "dataproc-worker@my-project.iam.gserviceaccount.com"
      service_account_scopes = [
        "cloud-platform",
      ]
    }
  }
}
