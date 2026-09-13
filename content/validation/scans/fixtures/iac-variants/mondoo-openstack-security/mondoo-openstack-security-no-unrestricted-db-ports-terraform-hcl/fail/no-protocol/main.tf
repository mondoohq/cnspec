# Non-compliant: the rule names no protocol, so Neutron matches every protocol, and the port range reaches MySQL on 3306.
resource "openstack_networking_secgroup_rule_v2" "mysql_any_proto" {
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_min    = 3306
  port_range_max    = 3306
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
