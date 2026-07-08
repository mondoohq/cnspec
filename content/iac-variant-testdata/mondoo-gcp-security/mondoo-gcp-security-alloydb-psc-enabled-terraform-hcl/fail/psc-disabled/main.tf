# Non-compliant: psc_config present but psc_enabled = false.
resource "google_alloydb_cluster" "fail_example" {
  cluster_id = "fail-cluster"
  location   = "us-central1"

  psc_config {
    psc_enabled = false
  }
}
