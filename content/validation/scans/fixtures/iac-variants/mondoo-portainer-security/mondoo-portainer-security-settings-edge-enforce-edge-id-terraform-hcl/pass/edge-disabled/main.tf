# Edge compute features are off, so enforce_edge_id is not required.
resource "portainer_settings" "this" {
  enable_edge_compute_features = false
}
