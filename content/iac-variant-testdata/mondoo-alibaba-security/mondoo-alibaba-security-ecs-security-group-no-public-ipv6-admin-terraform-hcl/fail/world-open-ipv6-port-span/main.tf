# An IPv6 rule whose range spans port 22 opens SSH to the whole IPv6 internet.
resource "alicloud_security_group_rule" "ipv6_span" {
  security_group_id = alicloud_security_group.app.id
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "1/1024"
  ipv6_cidr_ip      = "::/0"
  policy            = "accept"
  priority          = 1
}
