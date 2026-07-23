# A local Docker environment (type 1) is not an agent connection, so the
# TLS requirement does not apply even when tls_enabled is off.
resource "portainer_environment" "local" {
  name = "local"
  type = 1
}
