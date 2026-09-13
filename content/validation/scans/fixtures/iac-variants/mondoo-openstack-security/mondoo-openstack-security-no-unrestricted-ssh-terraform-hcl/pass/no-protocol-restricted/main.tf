# Compliant: the rule names no protocol and no port range, but it only admits an internal CIDR.
resource "openstack_networking_secgroup_rule_v2" "internal_any_proto" {
  direction         = "ingress"
  ethertype         = "IPv4"
  remote_ip_prefix  = "10.0.0.0/8"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
