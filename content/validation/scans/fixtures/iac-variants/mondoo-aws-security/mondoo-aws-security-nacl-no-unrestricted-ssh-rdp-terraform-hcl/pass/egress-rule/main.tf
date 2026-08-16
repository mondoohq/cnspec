# Compliant: this is an egress rule, so inbound SSH/RDP restrictions do not apply.
resource "aws_network_acl_rule" "pass_example" {
  network_acl_id = "acl-12345678"
  rule_number    = 100
  egress         = true
  protocol       = "tcp"
  rule_action    = "allow"
  cidr_block     = "0.0.0.0/0"
  from_port      = 22
  to_port        = 22
}
