# Compliant: egress via aws_security_group_rule.
resource "aws_security_group" "example" {
  name = "example"
}

resource "aws_security_group_rule" "example" {
  type              = "egress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  cidr_blocks       = ["10.0.0.0/8"]
  ipv6_cidr_blocks  = ["fd00::/8"]
  security_group_id = aws_security_group.example.id
}
