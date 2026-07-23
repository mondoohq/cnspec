# Compliant: port_security_enabled omitted, which defaults to enabled.
resource "openstack_networking_port_v2" "app" {
  name           = "app-port"
  network_id     = "a1b2c3d4-1234-5678-90ab-cdef01234567"
  admin_state_up = true
}
