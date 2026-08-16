resource "portainer_endpoint_settings" "prod" {
  endpoint_id = 1

  security_settings {
    allow_bind_mounts = false
  }
}
