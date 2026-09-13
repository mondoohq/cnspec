# Non-compliant: the rule names no protocol, so Neutron matches every protocol, and the port range reaches RDP on 3389.
resource "openstack_networking_secgroup_rule_v2" "rdp_any_proto" {
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_min    = 3389
  port_range_max    = 3389
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
