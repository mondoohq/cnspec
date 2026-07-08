resource "google_container_cluster" "primary" {
  name     = "primary"
  location = "us-central1"

  node_pool {
    name = "default-pool"

    upgrade_settings {
      strategy        = "SURGE"
      max_surge       = 1
      max_unavailable = 0
    }
  }
}
