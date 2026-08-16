# No internal_auth_settings block is declared, so no weak required length is set.
resource "portainer_settings" "this" {
  authentication_method = 2
}
