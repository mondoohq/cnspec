# Compliant: cluster has a psc_config block with psc_enabled = true.
resource "google_alloydb_cluster" "pass_example" {
  cluster_id = "pass-cluster"
  location   = "us-central1"

  psc_config {
    psc_enabled = true
  }
}
