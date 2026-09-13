# Non-compliant: a TCP rule with no port range covers every port, the orchestration ports included.
resource "openstack_networking_secgroup_rule_v2" "all_tcp" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
