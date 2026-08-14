resource "alicloud_security_group_rule" "ssh_from_office_v6" {
  security_group_id = alicloud_security_group.app.id
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "22/22"
  ipv6_cidr_ip      = "2001:db8:1::/48"
  policy            = "accept"
  priority          = 1
}
