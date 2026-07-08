resource "portainer_endpoint_settings" "prod" {
  endpoint_id                  = 1
  allow_container_capabilities = true
}
