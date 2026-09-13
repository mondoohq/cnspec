# Non-compliant: a 0-0 port range is how an absent range serializes, so it covers RDP on 3389.
resource "openstack_networking_secgroup_rule_v2" "all_tcp_zero" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 0
  port_range_max    = 0
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
