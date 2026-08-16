# A 1/65535 range from anywhere includes 22, so SSH is world-reachable even
# though the rule never names the port.
resource "alicloud_security_group_rule" "all_tcp_world" {
  security_group_id = alicloud_security_group.app.id
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "1/65535"
  cidr_ip           = "0.0.0.0/0"
  policy            = "accept"
  priority          = 1
}
