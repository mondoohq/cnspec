# Compliant: MySQL open only to an internal CIDR.
resource "openstack_networking_secgroup_rule_v2" "mysql" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 3306
  port_range_max    = 3306
  remote_ip_prefix  = "10.0.0.0/8"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
