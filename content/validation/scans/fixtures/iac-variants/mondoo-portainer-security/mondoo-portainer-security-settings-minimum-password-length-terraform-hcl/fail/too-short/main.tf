resource "portainer_settings" "this" {
  authentication_method = 1

  internal_auth_settings {
    required_password_length = 8
  }
}
