# Non-compliant: cluster block has no kms_key_name (Google-managed encryption).
resource "google_bigtable_instance" "fail_example" {
  name = "fail-instance"

  cluster {
    cluster_id   = "fail-cluster"
    zone         = "us-central1-b"
    num_nodes    = 3
    storage_type = "SSD"
  }
}
