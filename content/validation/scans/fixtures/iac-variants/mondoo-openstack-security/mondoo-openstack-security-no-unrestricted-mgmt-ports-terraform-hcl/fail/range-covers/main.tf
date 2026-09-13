# Non-compliant: wide TCP range that includes the Kubernetes API server port.
resource "openstack_networking_secgroup_rule_v2" "wide_tcp" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 6000
  port_range_max    = 7000
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
