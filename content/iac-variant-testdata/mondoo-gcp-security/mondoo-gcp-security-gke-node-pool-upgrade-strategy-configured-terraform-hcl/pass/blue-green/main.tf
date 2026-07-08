resource "google_container_cluster" "primary" {
  name     = "primary"
  location = "us-central1"

  node_pool {
    name = "default-pool"

    upgrade_settings {
      strategy = "BLUE_GREEN"

      blue_green_settings {
        node_pool_soak_duration = "3600s"

        standard_rollout_policy {
          batch_percentage    = 0.2
          batch_soak_duration = "60s"
        }
      }
    }
  }
}
