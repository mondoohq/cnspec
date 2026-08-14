resource "alicloud_security_group_rule" "rdp_from_bastion" {
  security_group_id = alicloud_security_group.windows.id
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "3389/3389"
  cidr_ip           = "10.0.100.0/24"
  policy            = "accept"
  priority          = 1
}
