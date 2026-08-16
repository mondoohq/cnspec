resource "portainer_endpoint_settings" "example" {
  endpoint_id       = portainer_environment.example.id
  allow_bind_mounts = false
}
