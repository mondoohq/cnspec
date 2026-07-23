resource "google_container_node_pool" "np" {
  name       = "np"
  cluster    = google_container_cluster.primary.id
  node_count = 3

  management {
    auto_repair  = false
    auto_upgrade = true
  }
}
