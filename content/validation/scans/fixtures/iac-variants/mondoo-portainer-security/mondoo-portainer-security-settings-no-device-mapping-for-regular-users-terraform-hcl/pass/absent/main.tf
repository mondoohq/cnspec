# allow_device_mapping is unset, so regular users cannot map host devices.
resource "portainer_endpoint_settings" "prod" {
  endpoint_id       = 1
  allow_bind_mounts = false
}
