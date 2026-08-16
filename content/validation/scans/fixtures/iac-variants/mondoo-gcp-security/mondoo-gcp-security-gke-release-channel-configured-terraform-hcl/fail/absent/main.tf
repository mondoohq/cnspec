# Non-compliant: no release_channel block configured.
resource "google_container_cluster" "primary" {
  name     = "no-channel-cluster"
  location = "us-central1"

  initial_node_count = 1
}
