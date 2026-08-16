resource "google_container_cluster" "alpha" {
  name                    = "alpha-cluster"
  location                = "us-central1"
  initial_node_count      = 1
  enable_kubernetes_alpha = true
}
