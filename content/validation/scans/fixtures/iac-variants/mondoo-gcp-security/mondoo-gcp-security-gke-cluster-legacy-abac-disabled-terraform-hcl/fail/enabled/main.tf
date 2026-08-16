resource "google_container_cluster" "abac" {
  name               = "abac-cluster"
  location           = "us-central1"
  initial_node_count = 1
  enable_legacy_abac = true
}
