resource "google_container_cluster" "primary" {
  name     = "primary"
  location = "us-central1"

  node_config {
    workload_metadata_config {
      mode = "GCE_METADATA"
    }
  }
}
