# required_password_length is unset; the check only flags an explicit weak value.
resource "portainer_settings" "this" {
  authentication_method = 2
}
