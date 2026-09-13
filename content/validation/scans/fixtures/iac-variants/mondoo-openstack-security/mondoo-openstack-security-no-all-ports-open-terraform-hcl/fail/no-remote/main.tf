# Non-compliant: no port range and no remote, so every port is reachable from
# any source.
resource "openstack_networking_secgroup_rule_v2" "wide_open_any_source" {
  direction         = "ingress"
  ethertype         = "IPv4"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
