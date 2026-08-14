resource "alicloud_security_group_rule" "https_public" {
  security_group_id = alicloud_security_group.app.id
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "443/443"
  cidr_ip           = "0.0.0.0/0"
  policy            = "accept"
  priority          = 1
}

resource "alicloud_security_group_rule" "internal_all" {
  security_group_id = alicloud_security_group.app.id
  type              = "ingress"
  ip_protocol       = "all"
  port_range        = "-1/-1"
  cidr_ip           = "10.0.0.0/8"
  policy            = "accept"
  priority          = 1
}
