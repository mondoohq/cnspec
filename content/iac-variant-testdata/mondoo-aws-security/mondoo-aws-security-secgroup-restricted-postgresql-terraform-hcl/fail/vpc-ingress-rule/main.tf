# Non-compliant: postgresql (5432) open to the world via aws_vpc_security_group_ingress_rule.
resource "aws_security_group" "postgresql" {
  name = "postgresql"
}

resource "aws_vpc_security_group_ingress_rule" "postgresql" {
  security_group_id = aws_security_group.postgresql.id
  from_port         = 5432
  to_port           = 5432
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}
