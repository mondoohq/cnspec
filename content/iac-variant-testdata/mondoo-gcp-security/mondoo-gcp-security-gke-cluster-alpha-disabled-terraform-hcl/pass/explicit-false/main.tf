resource "google_container_cluster" "primary" {
  name                    = "primary"
  location                = "us-central1"
  initial_node_count      = 1
  enable_kubernetes_alpha = false
}
