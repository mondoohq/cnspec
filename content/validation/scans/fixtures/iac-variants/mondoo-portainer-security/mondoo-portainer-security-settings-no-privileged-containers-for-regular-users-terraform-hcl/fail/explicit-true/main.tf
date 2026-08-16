resource "portainer_endpoint_settings" "example" {
  endpoint_id = portainer_environment.example.id

  security_settings {
    allow_privileged_mode = true
  }
}
