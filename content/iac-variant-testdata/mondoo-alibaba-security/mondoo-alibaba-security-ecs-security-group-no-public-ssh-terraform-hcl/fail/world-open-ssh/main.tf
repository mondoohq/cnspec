# SSH open to the entire internet.
resource "alicloud_security_group_rule" "ssh_world" {
  security_group_id = alicloud_security_group.app.id
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "22/22"
  cidr_ip           = "0.0.0.0/0"
  policy            = "accept"
  priority          = 1
}
