# Non-compliant: no psc_config block at all.
resource "google_alloydb_cluster" "fail_example" {
  cluster_id = "fail-cluster"
  location   = "us-central1"
}
