# Non-compliant: cluster block sets an empty kms_key_name.
resource "google_bigtable_instance" "fail_example" {
  name = "fail-instance"

  cluster {
    cluster_id   = "fail-cluster"
    zone         = "us-central1-b"
    num_nodes    = 3
    storage_type = "SSD"
    kms_key_name = ""
  }
}
