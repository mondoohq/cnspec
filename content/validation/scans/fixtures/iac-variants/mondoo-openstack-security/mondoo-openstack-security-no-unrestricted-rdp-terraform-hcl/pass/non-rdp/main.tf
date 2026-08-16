# Compliant: SSH only, no RDP.
resource "openstack_networking_secgroup_rule_v2" "ssh" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 22
  port_range_max    = 22
  remote_ip_prefix  = "10.0.0.0/8"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
