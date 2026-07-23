# authentication_method 1 is Portainer internal authentication.
resource "portainer_settings" "this" {
  authentication_method = 1
}
