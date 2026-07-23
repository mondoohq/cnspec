# Compliant: egress via aws_vpc_security_group_egress_rule.
resource "aws_security_group" "example" {
  name   = "example"
  vpc_id = "vpc-0123456789abcdef0"
}

resource "aws_vpc_security_group_egress_rule" "example" {
  security_group_id = aws_security_group.example.id
  ip_protocol       = "tcp"
  from_port         = 443
  to_port           = 443
  cidr_ipv4         = "10.0.0.0/8"
}
