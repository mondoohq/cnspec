# RDP open to the internet is among the most commonly brute-forced exposures.
resource "alicloud_security_group_rule" "rdp_world" {
  security_group_id = alicloud_security_group.windows.id
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "3389/3389"
  cidr_ip           = "0.0.0.0/0"
  policy            = "accept"
  priority          = 1
}
