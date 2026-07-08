# Non-compliant: ingress allow rule opens RDP (3389) to the entire internet.
resource "aws_network_acl_rule" "fail_example" {
  network_acl_id = "acl-12345678"
  rule_number    = 110
  egress         = false
  protocol       = "tcp"
  rule_action    = "allow"
  cidr_block     = "0.0.0.0/0"
  from_port      = 3389
  to_port        = 3389
}
