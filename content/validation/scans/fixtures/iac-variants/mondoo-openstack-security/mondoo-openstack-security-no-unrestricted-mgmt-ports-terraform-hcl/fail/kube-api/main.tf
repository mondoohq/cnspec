# Non-compliant: Kubernetes API 6443 exposed to the internet.
resource "openstack_networking_secgroup_rule_v2" "kube_api" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 6443
  port_range_max    = 6443
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
