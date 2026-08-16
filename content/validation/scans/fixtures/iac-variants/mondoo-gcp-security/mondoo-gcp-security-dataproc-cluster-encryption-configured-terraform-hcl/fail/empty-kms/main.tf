# Non-compliant: encryption_config present but kms_key_name is empty.
resource "google_dataproc_cluster" "fail_example" {
  name   = "analytics-cluster"
  region = "us-central1"

  cluster_config {
    encryption_config {
      kms_key_name = ""
    }
  }
}
