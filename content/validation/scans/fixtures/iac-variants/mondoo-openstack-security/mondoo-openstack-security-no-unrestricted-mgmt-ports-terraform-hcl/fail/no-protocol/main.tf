# Non-compliant: the rule names no protocol, so Neutron matches every protocol, and the port range reaches the Kubernetes API on 6443.
resource "openstack_networking_secgroup_rule_v2" "kube_api_any_proto" {
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_min    = 6443
  port_range_max    = 6443
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
