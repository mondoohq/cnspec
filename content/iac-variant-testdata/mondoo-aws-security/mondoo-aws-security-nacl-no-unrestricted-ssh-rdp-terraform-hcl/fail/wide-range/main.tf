# Non-compliant: a wide TCP port range (0-65535) open to the world spans 22 and 3389.
resource "aws_network_acl_rule" "wide" {
  network_acl_id = "acl-123"
  rule_number    = 110
  egress         = false
  protocol       = "tcp"
  rule_action    = "allow"
  cidr_block     = "0.0.0.0/0"
  from_port      = 0
  to_port        = 65535
}
