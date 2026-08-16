# The range is not written as 22/22, but it spans port 22, so SSH is open to the
# entire internet just the same.
resource "alicloud_security_group_rule" "ephemeral_span" {
  security_group_id = alicloud_security_group.app.id
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "20/25"
  cidr_ip           = "0.0.0.0/0"
  policy            = "accept"
  priority          = 1
}
