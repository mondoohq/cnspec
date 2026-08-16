# Compliant: the rule denies (rather than allows) SSH from the internet.
resource "aws_network_acl_rule" "pass_example" {
  network_acl_id = "acl-12345678"
  rule_number    = 100
  egress         = false
  protocol       = "tcp"
  rule_action    = "deny"
  cidr_block     = "0.0.0.0/0"
  from_port      = 22
  to_port        = 22
}
