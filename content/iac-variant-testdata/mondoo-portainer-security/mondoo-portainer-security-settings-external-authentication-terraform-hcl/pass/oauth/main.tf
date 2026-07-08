# authentication_method 3 is OAuth (external), not internal.
resource "portainer_settings" "this" {
  authentication_method = 3
}
