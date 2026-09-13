# Non-compliant: no remote_ip_prefix and no remote group, which Neutron treats
# as any source, so the Kubernetes API on 6443 is reachable from the internet.
resource "openstack_networking_secgroup_rule_v2" "kube_api_any_source" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 6443
  port_range_max    = 6443
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
