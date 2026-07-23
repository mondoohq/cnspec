resource "openstack_lb_listener_v2" "terminated" {
  name                      = "https-terminated"
  protocol                  = "TERMINATED_HTTPS"
  protocol_port             = 443
  loadbalancer_id           = "a1b2c3d4-1234-5678-90ab-cdef01234567"
  default_tls_container_ref = "https://barbican.example.com/v1/secrets/abcd1234"
  tls_versions              = ["TLSv1.0", "TLSv1.1", "TLSv1.2"]
}
