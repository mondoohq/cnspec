# Compliant: Dataproc cluster configures CMEK encryption for cluster data.
resource "google_dataproc_cluster" "pass_example" {
  name   = "analytics-cluster"
  region = "us-central1"

  cluster_config {
    encryption_config {
      kms_key_name = "projects/my-project/locations/us-central1/keyRings/dataproc/cryptoKeys/cluster-key"
    }
  }
}
