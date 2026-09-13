# Compliant: RDP/3389 reachable only from members of the bastion security group.
resource "openstack_networking_secgroup_rule_v2" "rdp_from_bastion" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 3389
  port_range_max    = 3389
  remote_group_id   = "c2d3e4f5-2345-6789-01bc-def012345678"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
