# allow_container_capabilities is unset, so regular users cannot add caps.
resource "portainer_endpoint_settings" "prod" {
  endpoint_id       = 1
  allow_bind_mounts = false
}
