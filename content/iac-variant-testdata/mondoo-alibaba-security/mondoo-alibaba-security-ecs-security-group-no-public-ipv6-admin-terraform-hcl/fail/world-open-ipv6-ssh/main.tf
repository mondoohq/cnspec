# The IPv6 equivalent of 0.0.0.0/0. Groups are often hardened on IPv4 only,
# leaving this path open.
resource "alicloud_security_group_rule" "ssh_world_v6" {
  security_group_id = alicloud_security_group.app.id
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "22/22"
  ipv6_cidr_ip      = "::/0"
  policy            = "accept"
  priority          = 1
}
