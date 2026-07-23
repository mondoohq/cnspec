# Non-compliant: allowed_cidrs permits the IPv6 wildcard.
resource "openstack_lb_listener_v2" "public" {
  name            = "app-listener"
  protocol        = "HTTPS"
  protocol_port   = 443
  loadbalancer_id = "a1b2c3d4-1234-5678-90ab-cdef01234567"
  allowed_cidrs   = ["::/0"]
}
