# Non-compliant: Dataproc cluster_config has no encryption_config block (no CMEK).
resource "google_dataproc_cluster" "fail_example" {
  name   = "analytics-cluster"
  region = "us-central1"

  cluster_config {
    master_config {
      num_instances = 1
      machine_type  = "n1-standard-4"
    }
  }
}
