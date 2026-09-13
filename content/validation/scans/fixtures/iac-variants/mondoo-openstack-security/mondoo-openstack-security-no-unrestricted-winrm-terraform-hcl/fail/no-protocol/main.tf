# Non-compliant: the rule names no protocol, so Neutron matches every protocol, and the port range reaches WinRM on 5985 and 5986.
resource "openstack_networking_secgroup_rule_v2" "winrm_any_proto" {
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_min    = 5985
  port_range_max    = 5986
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
