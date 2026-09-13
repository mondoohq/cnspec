# Non-compliant: no remote_ip_prefix and no remote group, which Neutron treats
# as any source, so RDP/3389 is reachable from the internet.
resource "openstack_networking_secgroup_rule_v2" "rdp_any_source" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 3389
  port_range_max    = 3389
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
