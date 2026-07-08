# allow_bind_mounts is unset, so regular users cannot mount host paths.
resource "portainer_endpoint_settings" "prod" {
  endpoint_id                  = 1
  allow_privileged_mode        = false
}
