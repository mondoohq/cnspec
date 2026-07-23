# Edge compute features are off, so trust_on_first_connect is not evaluated.
resource "portainer_settings" "this" {
  enable_edge_compute_features = false
}
