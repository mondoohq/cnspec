# Non-compliant: ingress allow rule opens SSH (22) to the entire internet.
resource "aws_network_acl_rule" "fail_example" {
  network_acl_id = "acl-12345678"
  rule_number    = 100
  egress         = false
  protocol       = "tcp"
  rule_action    = "allow"
  cidr_block     = "0.0.0.0/0"
  from_port      = 22
  to_port        = 22
}
