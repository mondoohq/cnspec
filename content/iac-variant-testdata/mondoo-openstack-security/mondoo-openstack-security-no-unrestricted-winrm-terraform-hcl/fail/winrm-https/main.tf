# Non-compliant: WinRM HTTPS/5986 exposed to the internet.
resource "openstack_networking_secgroup_rule_v2" "winrm_https" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 5986
  port_range_max    = 5986
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = "b1c2d3e4-1234-5678-90ab-cdef01234567"
}
