# Non-compliant: release_channel explicitly set to UNSPECIFIED (static version).
resource "google_container_cluster" "primary" {
  name     = "unspecified-channel-cluster"
  location = "us-central1"

  initial_node_count = 1

  release_channel {
    channel = "UNSPECIFIED"
  }
}
