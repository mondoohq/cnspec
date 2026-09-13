# Compliant: WinRM open only to a bastion CIDR.
resource "openstack_networking_secgroup_rule_v2" "winrm" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 5986
  port_range_max    = 5986
  remote_ip_prefix  = "203.0.113.10/32"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
