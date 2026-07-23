# Non-compliant: egress via aws_security_group_rule.
resource "aws_security_group" "example" {
  name = "example"
}

resource "aws_security_group_rule" "example" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  ipv6_cidr_blocks  = ["fd00::/8"]
  security_group_id = aws_security_group.example.id
}
