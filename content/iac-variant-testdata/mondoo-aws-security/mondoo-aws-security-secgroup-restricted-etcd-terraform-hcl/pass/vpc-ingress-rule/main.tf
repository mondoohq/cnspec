# Compliant: sensitive-port ingress via aws_vpc_security_group_ingress_rule.
resource "aws_security_group" "example" {
  name   = "example"
  vpc_id = "vpc-0123456789abcdef0"
}

resource "aws_vpc_security_group_ingress_rule" "example" {
  security_group_id = aws_security_group.example.id
  from_port         = 2379
  to_port           = 2380
  ip_protocol       = "tcp"
  cidr_ipv4         = "10.0.0.0/8"
}
