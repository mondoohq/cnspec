# Non-compliant: cluster has no encryption_config block (Google-managed keys).
resource "google_alloydb_cluster" "fail_example" {
  cluster_id = "fail-cluster"
  location   = "us-central1"
}
