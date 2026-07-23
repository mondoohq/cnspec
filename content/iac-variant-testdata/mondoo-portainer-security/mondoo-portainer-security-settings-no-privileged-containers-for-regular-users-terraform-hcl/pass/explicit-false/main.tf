resource "portainer_endpoint_settings" "example" {
  endpoint_id           = portainer_environment.example.id
  allow_privileged_mode = false
  allow_bind_mounts     = false
}
