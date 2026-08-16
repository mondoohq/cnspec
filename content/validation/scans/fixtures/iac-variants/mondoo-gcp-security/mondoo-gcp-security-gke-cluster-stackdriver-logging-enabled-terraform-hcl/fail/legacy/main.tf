resource "google_container_cluster" "legacy" {
  name               = "legacy-logging"
  location           = "us-central1"
  initial_node_count = 1
  logging_service    = "logging.googleapis.com"
}
