# Compliant: release_channel set to REGULAR.
resource "google_container_cluster" "primary" {
  name     = "channel-cluster"
  location = "us-central1"

  initial_node_count = 1

  release_channel {
    channel = "REGULAR"
  }
}
