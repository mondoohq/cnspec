resource "google_container_cluster" "primary" {
  name     = "primary"
  location = "us-central1"

  node_config {
    metadata = {
      disable-legacy-endpoints = "true"
    }
  }
}
