resource "portainer_settings" "this" {
  enable_edge_compute_features = true
  enforce_edge_id              = true
  trust_on_first_connect       = false
}
