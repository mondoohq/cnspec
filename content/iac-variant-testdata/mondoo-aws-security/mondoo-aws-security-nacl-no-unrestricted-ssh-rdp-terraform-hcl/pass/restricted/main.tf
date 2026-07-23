# Compliant: ingress allow rule for SSH is scoped to a private CIDR, not 0.0.0.0/0.
resource "aws_network_acl_rule" "pass_example" {
  network_acl_id = "acl-12345678"
  rule_number    = 100
  egress         = false
  protocol       = "tcp"
  rule_action    = "allow"
  cidr_block     = "10.0.0.0/16"
  from_port      = 22
  to_port        = 22
}
