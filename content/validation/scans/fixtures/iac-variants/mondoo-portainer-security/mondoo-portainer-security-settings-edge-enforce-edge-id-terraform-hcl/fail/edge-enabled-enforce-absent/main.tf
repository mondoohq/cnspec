# Edge compute is enabled but enforce_edge_id is left unset (defaults off).
resource "portainer_settings" "this" {
  enable_edge_compute_features = true
}
