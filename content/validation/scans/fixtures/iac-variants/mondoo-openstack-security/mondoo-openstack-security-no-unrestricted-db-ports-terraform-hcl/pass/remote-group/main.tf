# Compliant: MySQL/3306 reachable only from members of the application security group.
resource "openstack_networking_secgroup_rule_v2" "mysql_from_app" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 3306
  port_range_max    = 3306
  remote_group_id   = "c2d3e4f5-2345-6789-01bc-def012345678"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
