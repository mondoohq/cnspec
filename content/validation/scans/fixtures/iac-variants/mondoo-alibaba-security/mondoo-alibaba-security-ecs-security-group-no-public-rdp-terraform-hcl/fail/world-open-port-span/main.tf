# A broad privileged-port range that happens to span 3389 exposes RDP to the
# entire internet without ever naming the port.
resource "alicloud_security_group_rule" "wide_span" {
  security_group_id = alicloud_security_group.app.id
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "3300/3400"
  cidr_ip           = "0.0.0.0/0"
  policy            = "accept"
  priority          = 1
}
