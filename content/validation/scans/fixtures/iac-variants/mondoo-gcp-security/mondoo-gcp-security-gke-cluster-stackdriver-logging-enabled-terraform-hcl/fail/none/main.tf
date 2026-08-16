resource "google_container_cluster" "no_logging" {
  name               = "no-logging"
  location           = "us-central1"
  initial_node_count = 1
  logging_service    = "none"
}
