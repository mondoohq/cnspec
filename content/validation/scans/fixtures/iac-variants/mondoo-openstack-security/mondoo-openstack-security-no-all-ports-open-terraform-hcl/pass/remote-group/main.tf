# Compliant: an all-ports rule, but reachable only from members of the
# application security group.
resource "openstack_networking_secgroup_rule_v2" "all_from_app" {
  direction         = "ingress"
  ethertype         = "IPv4"
  remote_group_id   = "c2d3e4f5-2345-6789-01bc-def012345678"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
