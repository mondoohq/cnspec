# Compliant: listener is restricted to internal CIDRs.
resource "openstack_lb_listener_v2" "internal" {
  name            = "app-listener"
  protocol        = "HTTPS"
  protocol_port   = 443
  loadbalancer_id = "a1b2c3d4-1234-5678-90ab-cdef01234567"
  allowed_cidrs   = ["10.0.0.0/8", "192.168.0.0/16"]
}
