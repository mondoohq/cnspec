# Every protocol and every port, open to the internet.
resource "alicloud_security_group_rule" "all_world" {
  security_group_id = alicloud_security_group.app.id
  type              = "ingress"
  ip_protocol       = "all"
  port_range        = "-1/-1"
  cidr_ip           = "0.0.0.0/0"
  policy            = "accept"
  priority          = 1
}
